#!/usr/bin/env python3
"""
Veda Proxy - A JSON RPC service for various operations including file system access.
Communicates via stdin/stdout and can operate locally or remotely via SSH.
"""

import json
import sys
import stat
import pwd
import grp
import os
from pathlib import Path
from typing import Dict, List, Any, Optional, Tuple
import logging
import subprocess
import threading
import queue
import re

# Configure logging to file so it doesn't interfere with JSON RPC communication
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - %(message)s',
    filename='/tmp/veda-proxy.log',
    filemode='a'
)
logger = logging.getLogger(__name__)


class LSPManager:
    """Manages Language Server Protocol servers for different languages."""
    
    def __init__(self):
        self.servers = {}  # language -> {'process': process, 'request_id': int}
        self.message_buffers = {}  # language -> list of all LSP messages
        self.pending_requests = {}  # language -> {request_id: threading.Event}
        self.response_cache = {}  # language -> {request_id: response}
        self.diagnostics = {}  # uri -> list of diagnostics
        self.pending_file_opens = {}  # language -> set of file paths to open after init
        self.language_servers = {
            'go': {
                'command': ['gopls', '-logfile=/tmp/gopls.log', '-rpc.trace'],
                'extensions': ['.go'],
                'install_hint': 'go install golang.org/x/tools/gopls@latest'
            },
            'c': {
                'command': ['clangd'],
                'extensions': ['.c', '.h'],
                'install_hint': 'Install clangd from your package manager or https://clangd.llvm.org/'
            },
            'cpp': {
                'command': ['clangd'],
                'extensions': ['.cpp', '.cc', '.cxx', '.hpp', '.hh', '.hxx'],
                'install_hint': 'Install clangd from your package manager or https://clangd.llvm.org/'
            },
            'python': {
                'command': ['pylsp'],
                'extensions': ['.py', '.pyw'],
                'install_hint': 'Install python-lsp-server with: pip install python-lsp-server'
            }
        }
        
    def _find_project_root(self, start_path: Path) -> Path:
        """Find the project root by looking for various markers."""
        current = start_path
        root = current.root
        
        # Project root markers in order of preference
        markers = [
            '.git',           # Git repository
            'pyproject.toml', # Modern Python project
            'setup.py',       # Traditional Python project
            'setup.cfg',      # Alternative Python project config
            'Pipfile',        # Pipenv project
            'requirements.txt', # Basic Python project
            'package.json',   # Node.js project
            'go.mod',         # Go module
            'Cargo.toml',     # Rust project
            'Makefile',       # Make-based project
            '.vscode',        # VSCode workspace
            '.idea',          # IntelliJ workspace
        ]
        
        while current != current.parent:
            for marker in markers:
                if (current / marker).exists():
                    logger.info(f"Found project root at {current} (marker: {marker})")
                    return current
            current = current.parent
            
        # If no marker found, use the original path
        logger.info(f"No project root marker found, using {start_path}")
        return start_path
        
    def get_language_for_file(self, file_path: str) -> Optional[str]:
        """Determine the language based on file extension."""
        path = Path(file_path)
        ext = path.suffix.lower()
        
        for language, config in self.language_servers.items():
            if ext in config['extensions']:
                return language
        return None
        
    def start_server(self, language: str) -> bool:
        """Start the LSP server for the specified language."""
        if language in self.servers:
            logger.info(f"LSP server for {language} is already running")
            return True
            
        if language not in self.language_servers:
            logger.error(f"No LSP server configured for language: {language}")
            return False
            
        config = self.language_servers[language]
        cmd = config['command']
        
        try:
            logger.info(f"Starting {cmd[0]} for {language}")
            process = subprocess.Popen(
                cmd,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,  # Discard stderr since gopls logs to file
                text=True,
                bufsize=0  # Unbuffered
            )
            
            self.servers[language] = {
                'process': process,
                'request_id': 0,
                'initialized': False,
                'open_files': set()
            }
            
            # Initialize message buffer for this language
            self.message_buffers[language] = []
            self.pending_requests[language] = {}
            self.response_cache[language] = {}
            
            # Start a thread to continuously read messages from the LSP server
            def read_messages():
                while language in self.servers:
                    try:
                        msg = self._read_lsp_message(language)
                        if msg:
                            # Add to message buffer
                            self.message_buffers[language].append(msg)
                            
                            # Handle the message
                            if 'id' in msg and msg.get('id') in self.pending_requests[language]:
                                # This is a response to one of our requests
                                self.response_cache[language][msg['id']] = msg
                                self.pending_requests[language][msg['id']].set()
                            elif 'method' in msg:
                                # This is a notification or server request
                                self._handle_server_message(language, msg)
                    except Exception as e:
                        logger.error(f"Error reading LSP message: {e}")
                        break
            
            reader_thread = threading.Thread(target=read_messages, daemon=True)
            reader_thread.start()
            
            # Initialize the LSP connection
            if self._send_initialize(language):
                # Wait for initialization to complete
                self._wait_for_initialize(language)
                return True
            return False
            
        except FileNotFoundError:
            logger.error(f"{cmd[0]} not found. {config['install_hint']}")
            return False
        except Exception as e:
            logger.error(f"Failed to start {cmd[0]} for {language}: {e}")
            return False
    
    def start_server_async(self, language: str):
        """Start the LSP server asynchronously for the specified language."""
        if language in self.servers:
            logger.info(f"LSP server for {language} is already running")
            return
            
        if language not in self.language_servers:
            logger.error(f"No LSP server configured for language: {language}")
            return
            
        # Mark that initialization is in progress
        self.servers[language] = {'initializing': True}
        
        def init_server():
            try:
                # Remove the temporary marker
                if language in self.servers and self.servers[language].get('initializing'):
                    del self.servers[language]
                
                # Actually start the server
                if self.start_server(language):
                    logger.info(f"LSP server for {language} initialized asynchronously")
                    
                    # Process any pending file opens
                    if hasattr(self, 'pending_file_opens') and language in self.pending_file_opens:
                        pending_files = self.pending_file_opens[language]
                        for file_path in pending_files:
                            logger.info(f"Opening deferred file: {file_path}")
                            self._open_file(language, file_path)
                        del self.pending_file_opens[language]
            except Exception as e:
                logger.error(f"Failed to start LSP server for {language} asynchronously: {e}")
                # Clean up on failure
                if language in self.servers and self.servers[language].get('initializing'):
                    del self.servers[language]
        
        # Start initialization in a background thread
        init_thread = threading.Thread(target=init_server, daemon=True)
        init_thread.start()
    
    def stop_server(self, language: str) -> bool:
        """Stop the LSP server for the specified language."""
        if language not in self.servers:
            logger.warning(f"No LSP server running for language: {language}")
            return False
            
        server = self.servers[language]
        process = server['process']
        
        try:
            process.terminate()
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            logger.warning(f"LSP server for {language} did not terminate, killing it")
            process.kill()
            
        del self.servers[language]
        logger.info(f"LSP server for {language} stopped")
        return True
    
    def _send_initialize(self, language: str) -> bool:
        """Send the initialize request to the LSP server."""
        if language not in self.servers:
            return False
            
        server = self.servers[language]
        server['request_id'] += 1
        
        # Try to find the project root (look for git, pyproject.toml, setup.py, etc.)
        root_path = self._find_project_root(Path.cwd())
        
        request = {
            "jsonrpc": "2.0",
            "id": server['request_id'],
            "method": "initialize",
            "params": {
                "processId": os.getpid(),
                "rootUri": f"file://{root_path.absolute()}",
                "workspaceFolders": [{
                    "uri": f"file://{root_path.absolute()}",
                    "name": root_path.name
                }],
                "capabilities": {
                    "textDocument": {
                        "definition": {
                            "dynamicRegistration": False
                        },
                        "references": {
                            "dynamicRegistration": False
                        },
                        "synchronization": {
                            "dynamicRegistration": False,
                            "willSave": False,
                            "didSave": False,
                            "willSaveWaitUntil": False
                        }
                    }
                }
            }
        }
        self._send_message(language, request)
        return True
    
    def _wait_for_initialize(self, language: str):
        """Wait for the initialize response and send initialized notification."""
        # Wait for the initialize response
        server = self.servers[language]
        response = self._wait_for_response(language, server['request_id'])
        if response and "result" in response:
            logger.info(f"LSP server for {language} initialized successfully")
            
            # Send the initialized notification
            server['initialized'] = True
            
            notification = {
                "jsonrpc": "2.0",
                "method": "initialized",
                "params": {}
            }
            self._send_message(language, notification)
    
    def _send_message(self, language: str, message: Dict[str, Any]):
        """Send a message to an LSP server with proper LSP headers."""
        if language not in self.servers:
            return
            
        process = self.servers[language]['process']
        content = json.dumps(message)
        header = f"Content-Length: {len(content)}\r\n\r\n"
        full_message = header + content
        
        try:
            process.stdin.write(full_message)
            process.stdin.flush()
        except Exception as e:
            logger.error(f"Failed to send message to {language} LSP server: {e}")
    
    def get_definition(self, file_path: str, line: int, character: int) -> Optional[List[Dict[str, Any]]]:
        """Get the definition location for a symbol at the given position."""
        # Determine language from file extension
        language = self.get_language_for_file(file_path)
        if not language:
            logger.error(f"No language server configured for file: {file_path}")
            return None
            
        # Start server if not running (this shouldn't happen anymore since get_file starts it)
        if language not in self.servers:
            logger.warning(f"LSP server not running for {language}, starting it now")
            if not self.start_server(language):
                return None
                
        server = self.servers[language]
        file_uri = f"file://{Path(file_path).absolute()}"
        
        server['request_id'] += 1
        request_id = server['request_id']
        
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "textDocument/definition",
            "params": {
                "textDocument": {
                    "uri": file_uri
                },
                "position": {
                    "line": line,
                    "character": character
                }
            }
        }
        
        logger.info(f"Sending textDocument/definition request for {file_uri} at {line}:{character}")
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id)
        logger.info(f"Received response: {response}")
        if response and "result" in response:
            result = response["result"]
            # Normalize result to always be a list
            if isinstance(result, dict):
                return [result]
            elif isinstance(result, list):
                return result
            elif result is None:
                return []
            return result
        elif response and "error" in response:
            logger.error(f"LSP error: {response['error']}")
        return None
    
    def get_completions(self, file_path: str, line: int, character: int) -> Optional[Dict[str, Any]]:
        """Get completion suggestions at the given position."""
        # Determine language from file extension
        language = self.get_language_for_file(file_path)
        if not language:
            logger.error(f"No language server configured for file: {file_path}")
            return None
            
        # Ensure the server is started
        if language not in self.servers:
            logger.info(f"LSP server for {language} is not running, starting it...")
            if not self.start_server(language):
                logger.error(f"Failed to start LSP server for {language}")
                return None
            
        # Ensure the file is open in the LSP server
        self._open_file(language, file_path)
        
        # Prepare the completion request
        server = self.servers[language]
        server['request_id'] += 1
        request_id = server['request_id']
        # Convert to absolute path before creating URI
        abs_path = Path(file_path).absolute()
        file_uri = abs_path.as_uri()
        
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "textDocument/completion",
            "params": {
                "textDocument": {
                    "uri": file_uri
                },
                "position": {
                    "line": line,
                    "character": character
                },
                "context": {
                    "triggerKind": 1  # Invoked manually
                }
            }
        }
        
        logger.info(f"Sending textDocument/completion request for {file_uri} at {line}:{character}")
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id)
        logger.info(f"Received completion response: {response}")
        if response and "result" in response:
            return response["result"]
        elif response and "error" in response:
            logger.error(f"LSP error: {response['error']}")
        return None
    
    def get_references(self, file_path: str, line: int, character: int, include_declaration: bool = True) -> Optional[List[Dict[str, Any]]]:
        """Get all references to a symbol at the given position."""
        # Determine language from file extension
        language = self.get_language_for_file(file_path)
        if not language:
            logger.error(f"No language server configured for file: {file_path}")
            return None
            
        # Start server if not running
        if language not in self.servers:
            logger.warning(f"LSP server not running for {language}, starting it now")
            if not self.start_server(language):
                return None
                
        # Ensure file is open in LSP
        self._open_file(language, file_path)
                
        server = self.servers[language]
        file_uri = f"file://{Path(file_path).absolute()}"
        
        server['request_id'] += 1
        request_id = server['request_id']
        
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "textDocument/references",
            "params": {
                "textDocument": {
                    "uri": file_uri
                },
                "position": {
                    "line": line,
                    "character": character
                },
                "context": {
                    "includeDeclaration": include_declaration
                }
            }
        }
        
        logger.info(f"Sending textDocument/references request for {file_uri} at {line}:{character}")
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id)
        logger.info(f"Received references response: {response}")
        if response and "result" in response:
            result = response["result"]
            # Normalize result to always be a list
            if isinstance(result, list):
                return result
            elif result is None:
                return []
            return result
        elif response and "error" in response:
            logger.error(f"LSP error: {response['error']}")
        return None
    
    def get_signature_help(self, file_path: str, line: int, character: int) -> Optional[Dict[str, Any]]:
        """Get signature help for a function at the given position."""
        # Determine language from file extension
        language = self.get_language_for_file(file_path)
        if not language:
            logger.error(f"No language server configured for file: {file_path}")
            return None
            
        # Ensure the server is started
        if language not in self.servers:
            logger.info(f"LSP server for {language} is not running, starting it...")
            if not self.start_server(language):
                logger.error(f"Failed to start LSP server for {language}")
                return None
            
        # Ensure the file is open in the LSP server
        self._open_file(language, file_path)
        
        # Prepare the signature help request
        server = self.servers[language]
        server['request_id'] += 1
        request_id = server['request_id']
        # Convert to absolute path before creating URI
        abs_path = Path(file_path).absolute()
        file_uri = abs_path.as_uri()
        
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "textDocument/signatureHelp",
            "params": {
                "textDocument": {
                    "uri": file_uri
                },
                "position": {
                    "line": line,
                    "character": character
                }
            }
        }
        
        logger.info(f"Sending textDocument/signatureHelp request for {file_uri} at {line}:{character}")
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id)
        logger.info(f"Received signature help response: {response}")
        if response and "result" in response:
            return response["result"]
        elif response and "error" in response:
            logger.error(f"LSP error: {response['error']}")
        return None
    
    def get_document_symbols(self, file_path: str) -> Optional[List[Dict[str, Any]]]:
        """Get document symbols (functions, classes, etc.) for a file."""
        # Determine language from file extension
        ext = Path(file_path).suffix.lower()
        language = None
        for lang, config in self.language_servers.items():
            if ext in config['extensions']:
                language = lang
                break
                
        if not language:
            logger.info(f"No language server available for extension {ext}")
            return None
            
        # Start server if needed
        if not self.start_server(language):
            return None
            
        # Ensure file is open in LSP
        self._open_file(language, file_path)
        
        server = self.servers[language]
        file_uri = f"file://{Path(file_path).absolute()}"
        
        server['request_id'] += 1
        request_id = server['request_id']
        
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "textDocument/documentSymbol",
            "params": {
                "textDocument": {
                    "uri": file_uri
                }
            }
        }
        
        logger.info(f"Sending textDocument/documentSymbol request for {file_uri}")
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id)
        logger.info(f"Received documentSymbol response: {response}")
        
        if response and "result" in response:
            result = response["result"]
            if isinstance(result, list):
                return result
            elif result is None:
                return []
            return result
        elif response and "error" in response:
            logger.error(f"LSP error: {response['error']}")
        return None
    
    def get_workspace_symbols(self, query: str = "") -> Optional[List[Dict[str, Any]]]:
        """Get workspace symbols (functions, classes, etc.) across all files in the project."""
        # Collect symbols from all active language servers
        all_symbols = []
        
        for language in list(self.servers.keys()):
            if not self.servers[language]['initialized']:
                continue
                
            server = self.servers[language]
            server['request_id'] += 1
            request_id = server['request_id']
            
            request = {
                "jsonrpc": "2.0",
                "id": request_id,
                "method": "workspace/symbol",
                "params": {
                    "query": query
                }
            }
            
            logger.info(f"Sending workspace/symbol request to {language} server with query: {query}")
            self._send_message(language, request)
            
            # Wait for the response with our specific request ID
            response = self._wait_for_response(language, request_id)
            logger.info(f"Received workspace/symbol response from {language}: {response}")
            
            if response and "result" in response:
                result = response["result"]
                if isinstance(result, list):
                    # Add language info to each symbol
                    for symbol in result:
                        symbol['language'] = language
                    all_symbols.extend(result)
        
        # If no LSP servers are running, try to start them for common file types in the project
        if not all_symbols:
            # Find all source files in the project
            project_root = Path.cwd()
            for lang, config in self.language_servers.items():
                for ext in config['extensions']:
                    if list(project_root.rglob(f"*{ext}")):
                        logger.info(f"Found {ext} files, starting {lang} server")
                        if self.start_server(lang):
                            # Retry the query with the newly started server
                            server = self.servers[lang]
                            server['request_id'] += 1
                            request_id = server['request_id']
                            
                            request = {
                                "jsonrpc": "2.0",
                                "id": request_id,
                                "method": "workspace/symbol",
                                "params": {
                                    "query": query
                                }
                            }
                            
                            self._send_message(lang, request)
                            response = self._wait_for_response(lang, request_id)
                            
                            if response and "result" in response:
                                result = response["result"]
                                if isinstance(result, list):
                                    for symbol in result:
                                        symbol['language'] = lang
                                    all_symbols.extend(result)
        
        return all_symbols if all_symbols else None
    
    def get_folding_ranges(self, file_path: str) -> Optional[List[Dict[str, Any]]]:
        """Get folding ranges for a file."""
        # Determine language from file extension
        ext = Path(file_path).suffix.lower()
        language = None
        for lang, config in self.language_servers.items():
            if ext in config['extensions']:
                language = lang
                break
                
        if not language:
            logger.info(f"No language server available for extension {ext}")
            return None
            
        # Start server if needed
        if not self.start_server(language):
            return None
            
        # Ensure file is open in LSP
        self._open_file(language, file_path)
        
        server = self.servers[language]
        file_uri = f"file://{Path(file_path).absolute()}"
        
        server['request_id'] += 1
        request_id = server['request_id']
        
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "textDocument/foldingRange",
            "params": {
                "textDocument": {
                    "uri": file_uri
                }
            }
        }
        
        logger.info(f"Sending textDocument/foldingRange request for {file_uri}")
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id)
        logger.info(f"Received foldingRange response: {response}")
        
        if response and "result" in response:
            result = response["result"]
            if isinstance(result, list):
                return result
            elif result is None:
                return []
            return result
        elif response and "error" in response:
            logger.error(f"LSP error: {response['error']}")
        return None
    
    def _open_file(self, language: str, file_path: str):
        """Notify LSP server that a file has been opened."""
        if language not in self.servers:
            return
            
        # If server is still initializing, defer the file opening
        server = self.servers.get(language, {})
        if server.get('initializing'):
            # Queue the file to be opened once server is ready
            if not hasattr(self, 'pending_file_opens'):
                self.pending_file_opens = {}
            if language not in self.pending_file_opens:
                self.pending_file_opens[language] = set()
            self.pending_file_opens[language].add(file_path)
            logger.info(f"Deferring opening of {file_path} until {language} server is ready")
            return
            
        # Read file content
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
        except Exception as e:
            logger.error(f"Failed to read file {file_path}: {e}")
            return
            
        file_uri = f"file://{Path(file_path).absolute()}"
        
        # Detect language ID based on extension
        ext = Path(file_path).suffix.lower()
        language_id = {
            '.go': 'go',
            '.c': 'c',
            '.h': 'c',
            '.cpp': 'cpp',
            '.cc': 'cpp',
            '.cxx': 'cpp',
            '.hpp': 'cpp',
            '.hh': 'cpp',
            '.hxx': 'cpp',
            '.py': 'python',
            '.pyw': 'python'
        }.get(ext, 'plaintext')
        
        notification = {
            "jsonrpc": "2.0",
            "method": "textDocument/didOpen",
            "params": {
                "textDocument": {
                    "uri": file_uri,
                    "languageId": language_id,
                    "version": 1,
                    "text": content
                }
            }
        }
        
        self._send_message(language, notification)
    
    def _read_lsp_message(self, language: str) -> Optional[Dict[str, Any]]:
        """Read a single LSP message from the server."""
        if language not in self.servers:
            return None
            
        process = self.servers[language]['process']
        
        try:
            # Read headers
            headers = {}
            while True:
                line = process.stdout.readline()
                if line == '\r\n' or line == '\n':
                    break
                if ':' in line:
                    key, value = line.split(':', 1)
                    headers[key.strip()] = value.strip()
            
            # Read content
            if 'Content-Length' in headers:
                content_length = int(headers['Content-Length'])
                content = process.stdout.read(content_length)
                return json.loads(content)
                
        except Exception as e:
            logger.error(f"Failed to read message from {language} LSP server: {e}")
            
        return None
    
    def _handle_server_message(self, language: str, message: Dict[str, Any]):
        """Handle notifications and requests from the LSP server."""
        method = message.get('method', '')
        
        if method == 'window/showMessage':
            # This message should be shown to the user
            params = message.get('params', {})
            logger.info(f"LSP {language} showMessage: {params.get('message', '')}")
            # The editor will poll for these messages
        elif method == 'window/logMessage':
            # Just log these
            params = message.get('params', {})
            logger.debug(f"LSP {language}: {params.get('message', '')}")
        elif method == 'textDocument/publishDiagnostics':
            # Store diagnostics
            params = message.get('params', {})
            uri = params.get('uri', '')
            diagnostics = params.get('diagnostics', [])
            self.diagnostics[uri] = diagnostics
            logger.debug(f"Received {len(diagnostics)} diagnostics for {uri}")
    
    def _wait_for_response(self, language: str, request_id: int, timeout: float = 5.0) -> Optional[Dict[str, Any]]:
        """Wait for a response with a specific request ID."""
        if language not in self.pending_requests:
            return None
            
        event = threading.Event()
        self.pending_requests[language][request_id] = event
        
        # Wait for the response
        if event.wait(timeout):
            # Response received
            response = self.response_cache[language].pop(request_id, None)
            del self.pending_requests[language][request_id]
            return response
        else:
            # Timeout
            del self.pending_requests[language][request_id]
            logger.error(f"Timeout waiting for response to request {request_id}")
            return None
    
    def _read_response(self, language: str) -> Optional[Dict[str, Any]]:
        """Read a response from an LSP server (legacy method for compatibility)."""
        # This is now just a wrapper that reads one message
        return self._read_lsp_message(language)
    
    def get_message_buffer(self, language: str) -> List[Dict[str, Any]]:
        """Get all LSP messages for a specific language."""
        return self.message_buffers.get(language, [])
    
    def get_pending_notifications(self, language: str) -> List[Dict[str, Any]]:
        """Get pending showMessage notifications for a language."""
        notifications = []
        messages = self.message_buffers.get(language, [])
        
        for msg in messages:
            if msg.get('method') == 'window/showMessage' and not msg.get('shown', False):
                msg['shown'] = True  # Mark as shown
                notifications.append(msg)
        
        return notifications
    
    def stop_all_servers(self):
        """Stop all running LSP servers."""
        languages = list(self.servers.keys())
        for language in languages:
            self.stop_server(language)
    
    def get_all_diagnostics(self) -> Dict[str, List[Dict[str, Any]]]:
        """
        Get all diagnostics from all open files.
        
        Returns:
            Dictionary mapping URIs to lists of diagnostics
        """
        return self.diagnostics.copy()


class VedaProxy:
    """Main proxy class handling JSON RPC requests for various operations."""

    def __init__(self):
        self.lsp = LSPManager()
        self.methods = {
            'list_directory': self.list_directory,
            'list_directory_recursive': self.list_directory_recursive,
            'get_file': self.get_file,
            'write_file': self.write_file,
            'start_lsp_server': self.start_lsp_server,
            'stop_lsp_server': self.stop_lsp_server,
            'go_to_definition': self.go_to_definition,
            'get_lsp_messages': self.get_lsp_messages,
            'get_lsp_notifications': self.get_lsp_notifications,
            'get_document_symbols': self.get_document_symbols,
            'get_folding_ranges': self.get_folding_ranges,
            'get_signature_help': self.get_signature_help,
            'list_project_symbols': self.list_project_symbols,
            'find_references': self.find_references,
            'get_completions': self.get_completions,
            'get_diagnostics': self.get_diagnostics,
            'get_project_changes': self.get_project_changes,
        }

    def list_directory(self, path: str) -> List[Dict[str, Any]]:
        """
        List files and directories in the specified path with detailed information.

        Args:
            path: Directory path to list

        Returns:
            List of file/directory information dictionaries
        """
        directory_path = Path(path).expanduser().resolve()

        if not directory_path.exists():
            raise FileNotFoundError(f"Directory does not exist: {path}")

        if not directory_path.is_dir():
            raise NotADirectoryError(f"Path is not a directory: {path}")

        entries = []

        for entry in directory_path.iterdir():
            try:
                file_info = self._get_file_info(entry, directory_path)
                entries.append(file_info)
            except (OSError, PermissionError) as e:
                logger.warning(f"Could not get info for {entry}: {e}")
                # Add basic info even if we can't get all details
                entries.append({
                    'name': entry.name,
                    'path': str(entry.absolute()),
                    'type': 'unknown',
                    'error': str(e)
                })

        # Sort entries: directories first, then files, both alphabetically
        entries.sort(key=lambda x: (x.get('type') != 'directory', x['name'].lower()))

        return entries

    def list_directory_recursive(self, path: str) -> List[Dict[str, Any]]:
        """
        Recursively list all files and directories in the specified path and its subdirectories.
        
        Args:
            path: Directory path to start from
            
        Returns:
            List of file/directory information dictionaries with absolute paths
        """
        directory_path = Path(path).expanduser().resolve()
        
        if not directory_path.exists():
            raise FileNotFoundError(f"Directory does not exist: {path}")
            
        if not directory_path.is_dir():
            raise NotADirectoryError(f"Path is not a directory: {path}")
            
        entries = []
        
        # Directories to skip for performance and relevance
        skip_dirs = {
            '.git', 'node_modules', '.cache', '.local', '.npm', '.cargo',
            'target', 'dist', 'build', '__pycache__', '.venv', 'venv',
            '.mypy_cache', '.pytest_cache', '.tox', 'coverage', '.coverage'
        }
        
        def _walk_directory(current_path: Path):
                
            try:
                for entry in current_path.iterdir():
                    try:
                        # Skip certain directories
                        if entry.is_dir() and entry.name in skip_dirs:
                            continue
                            
                        # Create simplified file info for fuzzy listing
                        file_info = {
                            'name': entry.name,
                            'path': str(entry.absolute()),
                            'type': 'directory' if entry.is_dir() else 'file',
                            'size': entry.stat().st_size if entry.is_file() else 0
                        }
                        
                        entries.append(file_info)
                        
                        # Recurse into subdirectories
                        if entry.is_dir():
                            _walk_directory(entry)
                            
                    except (OSError, PermissionError) as e:
                        logger.warning(f"Could not access {entry}: {e}")
                        
            except (OSError, PermissionError) as e:
                logger.warning(f"Could not list directory {current_path}: {e}")
                
        _walk_directory(directory_path)
        
        # Sort by path for better organization
        entries.sort(key=lambda x: x['path'].lower())
        
        return entries

    def get_file(self, path: str) -> Dict[str, str]:
        """
        Read the contents of a file.
        
        Args:
            path: Absolute file path to read
            
        Returns:
            Dictionary with 'content' key containing the file contents
        """
        file_path = Path(path).expanduser().resolve()
        
        if not file_path.exists():
            raise FileNotFoundError(f"File does not exist: {path}")
            
        if not file_path.is_file():
            raise ValueError(f"Path is not a file: {path}")
            
        try:
            import time
            start_time = time.time()
            
            content = file_path.read_text(encoding='utf-8')
            read_time = time.time()
            logger.info(f"Read file content in {(read_time - start_time) * 1000:.1f}ms")
            
            # Notify LSP server about the opened file
            language = self.lsp.get_language_for_file(str(file_path))
            if language:
                # Start LSP server asynchronously if needed
                if language not in self.lsp.servers:
                    logger.info(f"Starting LSP server for {language} asynchronously when opening {path}")
                    self.lsp.start_server_async(language)
                
                # Notify LSP about the opened file (will be deferred if server is still initializing)
                self.lsp._open_file(language, str(file_path))
                logger.info(f"Requested LSP notification for opened file: {path}")
            
            lsp_time = time.time()
            logger.info(f"LSP operations took {(lsp_time - read_time) * 1000:.1f}ms")
            
            # Get current git branch
            branch = "main"  # default
            try:
                cmd_branch = ['git', 'rev-parse', '--abbrev-ref', 'HEAD']
                result_branch = subprocess.run(cmd_branch, capture_output=True, text=True, cwd=file_path.parent)
                if result_branch.returncode == 0:
                    branch = result_branch.stdout.strip()
            except Exception as e:
                logger.warning(f"Could not get git branch: {e}")
            
            branch_time = time.time()
            logger.info(f"Git branch detection took {(branch_time - lsp_time) * 1000:.1f}ms")
            
            # Get git changes for the file
            changes = {'added': [], 'deleted': [], 'modified': []}
            try:
                # Run git diff for the file
                cmd = ['git', 'diff', 'HEAD', '--', str(file_path)]
                result = subprocess.run(cmd, capture_output=True, text=True, cwd=file_path.parent)
                
                if result.returncode != 0:
                    # Try unstaged changes
                    cmd = ['git', 'diff', '--', str(file_path)]
                    result = subprocess.run(cmd, capture_output=True, text=True, cwd=file_path.parent)
                
                if result.returncode == 0 and result.stdout:
                    changes = self._parse_git_diff(result.stdout)
                
                # Also check staged changes
                cmd_staged = ['git', 'diff', '--cached', '--', str(file_path)]
                result_staged = subprocess.run(cmd_staged, capture_output=True, text=True, cwd=file_path.parent)
                
                if result_staged.returncode == 0 and result_staged.stdout:
                    staged_changes = self._parse_git_diff(result_staged.stdout)
                    # Merge staged and unstaged changes
                    for key in ['added', 'deleted', 'modified']:
                        changes[key] = sorted(list(set(changes.get(key, []) + staged_changes.get(key, []))))
            except Exception as e:
                logger.warning(f"Could not get git changes for {path}: {e}")
                # Don't fail the whole operation if git diff fails
            
            git_time = time.time()
            logger.info(f"Git diff operations took {(git_time - branch_time) * 1000:.1f}ms")
            logger.info(f"Total get_file time: {(git_time - start_time) * 1000:.1f}ms")
            
            return {
                'content': content,
                'changes': changes,
                'branch': branch
            }
        except UnicodeDecodeError:
            # Try reading as binary and return error
            raise ValueError(f"File is not a text file: {path}")
        except PermissionError:
            raise PermissionError(f"Permission denied: {path}")

    def write_file(self, path: str, content: str) -> Dict[str, bool]:
        """
        Write content to a file.
        
        Args:
            path: Absolute file path to write
            content: Content to write to the file
            
        Returns:
            Dictionary with 'success' key indicating if the write was successful
        """
        file_path = Path(path).expanduser().resolve()
        logger.info(f"Writing file: {path} (resolved to {file_path})")
        
        try:
            # Create parent directories if they don't exist
            file_path.parent.mkdir(parents=True, exist_ok=True)
            
            # Write the content
            file_path.write_text(content, encoding='utf-8')
            
            # Notify LSP server about the file change
            language = self.lsp.get_language_for_file(str(file_path))
            if language and language in self.lsp.servers:
                # TODO: Send didSave notification to LSP server
                logger.info(f"File saved, should notify LSP server for {language}")
            
            return {'success': True}
        except PermissionError:
            raise PermissionError(f"Permission denied: {path}")
        except Exception as e:
            logger.error(f"Failed to write file {path}: {str(e)}")
            raise

    def _get_file_info(self, path: Path, base_path: Path) -> Dict[str, Any]:
        """
        Internal method to get detailed file information.

        Args:
            path: Path object
            base_path: Base path to calculate relative paths from (required)

        Returns:
            Dictionary with file information
        """
        stat_info = path.stat()

        # Determine file type
        if path.is_dir():
            file_type = 'directory'
        elif path.is_file():
            file_type = 'file'
        elif path.is_symlink():
            file_type = 'symlink'
        elif stat.S_ISBLK(stat_info.st_mode):
            file_type = 'block_device'
        elif stat.S_ISCHR(stat_info.st_mode):
            file_type = 'char_device'
        elif stat.S_ISFIFO(stat_info.st_mode):
            file_type = 'fifo'
        elif stat.S_ISSOCK(stat_info.st_mode):
            file_type = 'socket'
        else:
            file_type = 'unknown'

        # Get permissions in octal and human-readable format
        permissions_octal = oct(stat_info.st_mode)[-3:]
        permissions_str = stat.filemode(stat_info.st_mode)

        # Get owner and group names
        try:
            owner_name = pwd.getpwuid(stat_info.st_uid).pw_name
        except KeyError:
            owner_name = str(stat_info.st_uid)

        try:
            group_name = grp.getgrgid(stat_info.st_gid).gr_name
        except KeyError:
            group_name = str(stat_info.st_gid)

        # Handle symlinks
        symlink_target = None
        if path.is_symlink():
            try:
                symlink_target = str(path.readlink())
            except OSError:
                symlink_target = None

        return {
            'name': path.name,
            'path': str(path.absolute()),
            'type': file_type,
            'size': stat_info.st_size,
            'permissions': {
                'octal': permissions_octal,
                'string': permissions_str
            },
            'owner': {
                'uid': stat_info.st_uid,
                'name': owner_name
            },
            'group': {
                'gid': stat_info.st_gid,
                'name': group_name
            },
            'timestamps': {
                'modified': stat_info.st_mtime,
                'accessed': stat_info.st_atime,
                'created': stat_info.st_ctime
            },
            'symlink_target': symlink_target
        }
    
    def start_lsp_server(self, language: str) -> Dict[str, Any]:
        """Start the LSP server for the specified language."""
        success = self.lsp.start_server(language)
        return {'success': success}
    
    def stop_lsp_server(self, language: str) -> Dict[str, Any]:
        """Stop the LSP server for the specified language."""
        success = self.lsp.stop_server(language)
        return {'success': success}
    
    def go_to_definition(self, file_path: str, line: int, character: int) -> Dict[str, Any]:
        """Get the definition location for a symbol at the given position."""
        try:
            result = self.lsp.get_definition(file_path, line, character)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No definition found'}
        except Exception as e:
            logger.error(f"Error in go_to_definition: {e}")
            return {'success': False, 'error': str(e)}
    
    def get_lsp_messages(self, language: str) -> Dict[str, Any]:
        """Get all LSP messages for a specific language."""
        messages = self.lsp.get_message_buffer(language)
        return {'success': True, 'messages': messages}
    
    def get_lsp_notifications(self, language: str) -> Dict[str, Any]:
        """Get pending showMessage notifications for a language."""
        notifications = self.lsp.get_pending_notifications(language)
        return {'success': True, 'notifications': notifications}
    
    def get_document_symbols(self, file_path: str) -> Dict[str, Any]:
        """Get document symbols (functions, classes, etc.) for a file."""
        try:
            result = self.lsp.get_document_symbols(file_path)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No symbols found'}
        except Exception as e:
            logger.error(f"Error in get_document_symbols: {e}")
            return {'success': False, 'error': str(e)}
    
    def get_folding_ranges(self, file_path: str) -> Dict[str, Any]:
        """Get folding ranges for a file."""
        try:
            result = self.lsp.get_folding_ranges(file_path)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No folding ranges found'}
        except Exception as e:
            logger.error(f"Error in get_folding_ranges: {e}")
            return {'success': False, 'error': str(e)}
    
    def get_signature_help(self, file_path: str, line: int, character: int) -> Dict[str, Any]:
        """Get signature help for a function at the given position."""
        try:
            result = self.lsp.get_signature_help(file_path, line, character)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No signature help available'}
        except Exception as e:
            logger.error(f"Error in get_signature_help: {e}")
            return {'success': False, 'error': str(e)}
    
    def list_project_symbols(self, query: str = "") -> Dict[str, Any]:
        """Get all symbols across the entire project workspace."""
        try:
            result = self.lsp.get_workspace_symbols(query)
            if result is not None:
                # Format symbols for the fuzzy list widget
                formatted_symbols = []
                for symbol in result:
                    # Extract file path from URI
                    uri = symbol.get('location', {}).get('uri', '')
                    if uri.startswith('file://'):
                        file_path = uri[7:]  # Remove 'file://' prefix
                    else:
                        file_path = uri
                    
                    # Format display name with symbol kind and file
                    kind_names = {
                        1: "File", 2: "Module", 3: "Namespace", 4: "Package",
                        5: "Class", 6: "Method", 7: "Property", 8: "Field",
                        9: "Constructor", 10: "Enum", 11: "Interface", 12: "Function",
                        13: "Variable", 14: "Constant", 15: "String", 16: "Number",
                        17: "Boolean", 18: "Array", 19: "Object", 20: "Key",
                        21: "Null", 22: "EnumMember", 23: "Struct", 24: "Event",
                        25: "Operator", 26: "TypeParameter"
                    }
                    kind = symbol.get('kind', 0)
                    kind_name = kind_names.get(kind, f"Unknown({kind})")
                    
                    name = symbol.get('name', '')
                    # Don't add decorations here - Go will add icons
                    display_name = f"{name} - {file_path}"
                    
                    formatted_symbol = {
                        'name': display_name,
                        'raw_name': name,  # Raw name for Go to format with icons
                        'path': file_path,
                        'line': symbol.get('location', {}).get('range', {}).get('start', {}).get('line', 0),
                        'character': symbol.get('location', {}).get('range', {}).get('start', {}).get('character', 0),
                        'kind': kind,
                        'language': symbol.get('language', 'unknown')
                    }
                    formatted_symbols.append(formatted_symbol)
                
                return {'success': True, 'symbols': formatted_symbols}
            return {'success': False, 'error': 'No symbols found in project'}
        except Exception as e:
            logger.error(f"Error in list_project_symbols: {e}")
            return {'success': False, 'error': str(e)}
    
    def find_references(self, file_path: str, line: int, character: int, include_declaration: bool = True) -> Dict[str, Any]:
        """Find all references to a symbol at the given position."""
        try:
            result = self.lsp.get_references(file_path, line, character, include_declaration)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No references found'}
        except Exception as e:
            logger.error(f"Error in find_references: {e}")
            return {'success': False, 'error': str(e)}
    
    def get_completions(self, file_path: str, line: int, character: int) -> Dict[str, Any]:
        """Get completion suggestions at the given position."""
        try:
            result = self.lsp.get_completions(file_path, line, character)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No completions found'}
        except Exception as e:
            logger.error(f"Error in get_completions: {e}")
            return {'success': False, 'error': str(e)}
    
    def get_diagnostics(self) -> Dict[str, Any]:
        """Get all diagnostics from all open files."""
        try:
            diagnostics = self.lsp.get_all_diagnostics()
            return {'success': True, 'diagnostics': diagnostics}
        except Exception as e:
            logger.error(f"Error in get_diagnostics: {e}")
            return {'success': False, 'error': str(e)}
    
    def get_project_changes(self) -> Dict[str, Any]:
        """
        Get all changed files in the project with their diff information.
        """
        try:
            # Get list of changed files
            cmd = ['git', 'diff', '--name-status', 'HEAD']
            result = subprocess.run(cmd, capture_output=True, text=True)
            
            unstaged_files = {}
            if result.returncode == 0:
                for line in result.stdout.strip().split('\n'):
                    if line:
                        parts = line.split('\t')
                        if len(parts) >= 2:
                            status, filepath = parts[0], parts[1]
                            unstaged_files[filepath] = status
            
            # Get staged files
            cmd_staged = ['git', 'diff', '--cached', '--name-status']
            result_staged = subprocess.run(cmd_staged, capture_output=True, text=True)
            
            staged_files = {}
            if result_staged.returncode == 0:
                for line in result_staged.stdout.strip().split('\n'):
                    if line:
                        parts = line.split('\t')
                        if len(parts) >= 2:
                            status, filepath = parts[0], parts[1]
                            staged_files[filepath] = status
            
            # Combine all changes
            all_files = {}
            for f, status in unstaged_files.items():
                all_files[f] = {'status': status, 'staged': False}
            for f, status in staged_files.items():
                if f in all_files:
                    all_files[f]['staged'] = True
                else:
                    all_files[f] = {'status': status, 'staged': True}
            
            # Get detailed changes for each file
            changes = []
            for filepath, info in all_files.items():
                try:
                    # Get the diff for this file
                    if info['staged']:
                        cmd = ['git', 'diff', '--cached', '--', filepath]
                    else:
                        cmd = ['git', 'diff', 'HEAD', '--', filepath]
                    
                    result = subprocess.run(cmd, capture_output=True, text=True)
                    if result.returncode == 0:
                        diff_info = self._parse_git_diff(result.stdout)
                        changes.append({
                            'file': filepath,
                            'status': info['status'],
                            'staged': info['staged'],
                            'added_lines': diff_info.get('added', []),
                            'deleted_lines': diff_info.get('deleted', []),
                            'hunks': diff_info.get('hunks', [])
                        })
                except Exception as e:
                    logger.error(f"Error processing {filepath}: {e}")
            
            return {'success': True, 'changes': changes}
            
        except Exception as e:
            logger.error(f"Error in get_project_changes: {e}")
            return {'success': False, 'error': str(e)}
    
    def _parse_git_diff(self, diff_output: str) -> Dict[str, Any]:
        """
        Parse git diff output to extract line numbers of changes.
        """
        added_lines = []
        deleted_lines = []
        modified_lines = []
        hunks = []
        
        current_old_line = 0
        current_new_line = 0
        
        for line in diff_output.split('\n'):
            # Parse hunk header: @@ -old_start,old_count +new_start,new_count @@
            hunk_match = re.match(r'^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@', line)
            if hunk_match:
                old_start = int(hunk_match.group(1))
                new_start = int(hunk_match.group(3))
                current_old_line = old_start
                current_new_line = new_start
                
                # Store hunk info
                hunks.append({
                    'old_start': old_start,
                    'new_start': new_start,
                    'header': line
                })
                continue
            
            # Process diff lines
            if line.startswith('+') and not line.startswith('+++'):
                # Added line
                added_lines.append(current_new_line)
                current_new_line += 1
            elif line.startswith('-') and not line.startswith('---'):
                # Deleted line (mark it at the next line position)
                deleted_lines.append(current_old_line)
                current_old_line += 1
            elif line.startswith(' '):
                # Unchanged line
                current_old_line += 1
                current_new_line += 1
        
        # Mark modified lines (lines that have both additions and deletions nearby)
        # This is a simple heuristic - can be improved
        deleted_set = set(deleted_lines)
        for added in added_lines:
            if added in deleted_set or (added - 1) in deleted_set:
                modified_lines.append(added)
        
        return {
            'added': sorted(list(set(added_lines) - set(modified_lines))),
            'deleted': sorted(deleted_lines),
            'modified': sorted(modified_lines),
            'hunks': hunks
        }


class JSONRPCHandler:
    """Handles JSON RPC communication over stdin/stdout."""

    def __init__(self, proxy: VedaProxy):
        self.proxy = proxy

    def handle_request(self, request_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Handle a single JSON RPC request.

        Args:
            request_data: Parsed JSON RPC request

        Returns:
            JSON RPC response dictionary
        """
        try:
            method_name = request_data.get('method')
            params = request_data.get('params', {})
            request_id = request_data.get('id')

            if not method_name or method_name not in self.proxy.methods:
                return {
                    'id': request_id,
                    'error': f"Unknown method: {method_name}"
                }

            # Call the method
            if isinstance(params, dict):
                result = self.proxy.methods[method_name](**params)
            elif isinstance(params, list):
                result = self.proxy.methods[method_name](*params)
            else:
                result = self.proxy.methods[method_name](params)

            return {
                'id': request_id,
                'result': result
            }

        except Exception as e:
            return {
                'id': request_data.get('id'),
                'error': str(e)
            }

    def run(self):
        """Main loop to handle JSON RPC requests from stdin."""
        logger.info("Veda Proxy started")

        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue

            try:
                request_data = json.loads(line)
                response = self.handle_request(request_data)
                response_json = json.dumps(response)
                logger.info(f"Sending response: {response_json[:200]}...")  # Log first 200 chars
                print(response_json, flush=True)

            except json.JSONDecodeError:
                print(json.dumps({'error': 'Invalid JSON'}), flush=True)


def main():
    """Main entry point."""
    proxy = VedaProxy()
    handler = JSONRPCHandler(proxy)
    
    try:
        handler.run()
    except KeyboardInterrupt:
        logger.info("Shutting down")
    finally:
        # Clean up all LSP servers
        proxy.lsp.stop_all_servers()


if __name__ == '__main__':
    main()
