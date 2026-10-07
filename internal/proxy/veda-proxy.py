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
import difflib
import logging
import subprocess
import threading
import queue
import re
import pty
import selectors
import os
import fcntl
import termios
import base64
import uuid
import struct
import time
import argparse
from urllib.parse import urlparse, quote

# Proxy version must match the Go client's ProxyVersion constant.
PROXY_VERSION = "1.0.52"

# Set up logging directory
from datetime import datetime
cache_dir = os.path.expanduser('~/.cache/veda')
os.makedirs(cache_dir, exist_ok=True)  # Ensure base cache directory exists
log_dir = os.path.join(cache_dir, 'logs')
os.makedirs(log_dir, exist_ok=True)  # Ensure logs subdirectory exists

# Generate log filename with timestamp
timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
log_file = os.path.join(log_dir, f'veda-proxy_{timestamp}.log')

# Configure logging to file so it doesn't interfere with JSON RPC communication
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - %(message)s',
    filename=log_file,
    filemode='a'  # Append mode instead of truncate
)
logger = logging.getLogger(__name__)


class LSPManager:
    """Manages Language Server Protocol servers for different languages."""

    def __init__(self, test_lsp_overrides=None):
        self.servers = {}  # language -> {'process': process, 'request_id': int}
        self.message_buffers = {}  # language -> list of all LSP messages
        self.pending_requests = {}  # language -> {request_id: threading.Event}
        self.response_cache = {}  # language -> {request_id: response}
        self.request_details = {}  # language -> {request_id: {'method': str, 'file': str, 'timestamp': float}}
        self.diagnostics = {}  # uri -> list of diagnostics
        self.pending_file_opens = {}  # language -> set of file paths to open after init
        self.file_versions = {}  # file_path -> version number for LSP synchronization
        self.lsp_errors = {}  # language -> error message when LSP server fails to start
        self.language_servers = {
            'go': {
                'command': ['gopls', '-logfile=/tmp/gopls.log', '-rpc.trace'],
                'extensions': ['.go'],
                'install_hint': 'go install golang.org/x/tools/gopls@latest'
            },
            'c': {
                'command': ['clangd', '--log=verbose', '--pretty'],
                'extensions': ['.c', '.h', '.m'],
                'install_hint': 'Install clangd from your package manager or https://clangd.llvm.org/'
            },
            'cpp': {
                'command': ['clangd', '--log=verbose', '--pretty'],
                'extensions': ['.cpp', '.cc', '.cxx', '.hpp', '.hh', '.hxx'],
                'install_hint': 'Install clangd from your package manager or https://clangd.llvm.org/'
            },
            'python': {
                'command': ['pylsp'],
                'extensions': ['.py', '.pyw'],
                'install_hint': 'Install python-lsp-server with: pip install python-lsp-server'
            },
            'javascript': {
                'command': ['typescript-language-server', '--stdio'],
                'extensions': ['.js', '.jsx', '.mjs', '.cjs'],
                'install_hint': 'Install with: npm i -g typescript typescript-language-server'
            },
            'typescript': {
                'command': ['typescript-language-server', '--stdio'],
                'extensions': ['.ts', '.tsx', '.mts', '.cts'],
                'install_hint': 'Install with: npm i -g typescript typescript-language-server'
            },
            'bash': {
                'command': ['bash-language-server', 'start'],
                'extensions': ['.sh', '.bash'],
                'install_hint': 'Install with: npm i -g bash-language-server'
            }
        }

        # Apply test overrides if provided
        if test_lsp_overrides:
            for lang, cmd in test_lsp_overrides.items():
                if lang in self.language_servers:
                    self.language_servers[lang]['command'] = [cmd]
                    logger.info(f"Test override: Using '{cmd}' for {lang} LSP server")
        
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
            # If it's just the initializing marker, proceed to start the server
            if self.servers[language].get('initializing'):
                pass  # Continue with server startup
            else:
                logger.info(f"LSP server for {language} is already running")
                return True
            
        if language not in self.language_servers:
            logger.error(f"No LSP server configured for language: {language}")
            return False
            
        config = self.language_servers[language]
        cmd = config['command']
        
        try:
            logger.info(f"Starting {language} LSP server: {cmd[0]}")
            # Route clangd stderr to a dedicated log file for diagnostics
            stderr_target = subprocess.DEVNULL
            try:
                if len(cmd) > 0 and os.path.basename(cmd[0]) == 'clangd':
                    from datetime import datetime as _dt
                    clangd_log = os.path.join(log_dir, f"clangd_{_dt.now().strftime('%Y%m%d_%H%M%S')}.log")
                    fh = open(clangd_log, 'a')
                    stderr_target = fh
            except Exception as _e:
                logger.warning(f"Failed to prepare clangd stderr log: {_e}")

            process = subprocess.Popen(
                cmd,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=stderr_target,
                text=True,
                bufsize=0  # Unbuffered
            )
            
            # Preserve 'initializing' flag set by start_server_async so that
            # _open_file defers didOpen until after the initialized notification.
            was_initializing = self.servers.get(language, {}).get('initializing', False)
            self.servers[language] = {
                'process': process,
                'request_id': 0,
                'initialized': False,
                'initializing': was_initializing,
                'open_files': set(),
                'file_versions': {},  # file_path -> version number
                # Keep handle to stderr log (if any) so we can close it on shutdown
                'stderr_log': (fh if 'fh' in locals() else None)
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
                        if not msg:
                            # Server closed or no more data; cleanly stop and exit the loop
                            try:
                                self.stop_server(language)
                            except Exception:
                                # Ensure we don't spin/log on a dead server
                                if language in self.servers:
                                    del self.servers[language]
                            break

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
                        logger.error(f"Error reading {language} LSP message: {e}", exc_info=True)
                        # Don't break - try to continue reading
                        # This prevents one bad message from killing the reader thread
                        import time
                        time.sleep(0.1)  # Brief pause before retrying
            
            reader_thread = threading.Thread(target=read_messages, daemon=True)
            reader_thread.start()
            
            # Initialize the LSP connection
            if self._send_initialize(language):
                # Wait for initialization to complete
                self._wait_for_initialize(language)
                # Check if initialization actually succeeded
                if language in self.servers and self.servers[language].get('initialized'):
                    return True
                else:
                    # Server didn't initialize properly (possibly exited immediately)
                    logger.error(f"LSP server for {language} failed to initialize")
                    return False
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
                # Actually start the server (will replace the initializing marker)
                if self.start_server(language):
                    logger.info(f"LSP server for {language} initialized asynchronously")
                    # Clear any previous error for this language
                    if language in self.lsp_errors:
                        del self.lsp_errors[language]

                    # Process any pending file opens
                    if hasattr(self, 'pending_file_opens') and language in self.pending_file_opens:
                        pending_files = self.pending_file_opens[language]
                        for file_path in pending_files:
                            logger.info(f"Opening deferred file: {file_path}")
                            self._open_file(language, file_path)
                        del self.pending_file_opens[language]

                    # Notify editor that LSP is ready
                    self._notify_editor_lsp_status(language, "ready")
                else:
                    # start_server returned False - server failed to start
                    error_msg = f"LSP server for {language} failed to start"
                    self.lsp_errors[language] = error_msg
                    logger.error(error_msg)
                    self._notify_editor_lsp_status(language, "error", error_msg)
            except Exception as e:
                logger.error(f"Failed to start LSP server for {language} asynchronously: {e}")
                # Store the error message
                error_msg = str(e)
                self.lsp_errors[language] = error_msg
                # Clean up on failure
                if language in self.servers and self.servers[language].get('initializing'):
                    del self.servers[language]
                self._notify_editor_lsp_status(language, "error", error_msg)
        
        # Start initialization in a background thread
        init_thread = threading.Thread(target=init_server, daemon=True)
        init_thread.start()
    
    def stop_server(self, language: str) -> bool:
        """Stop the LSP server for the specified language."""
        if language not in self.servers:
            logger.warning(f"No LSP server running for language: {language}")
            return False

        server = self.servers[language]
        process = server.get('process')

        # If server was never fully initialized, record an error and notify editor
        if not server.get('initialized'):
            error_msg = f"LSP server for {language} exited unexpectedly"
            self.lsp_errors[language] = error_msg
            self._notify_editor_lsp_status(language, "error", error_msg)

        if process:
            try:
                process.terminate()
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                logger.warning(f"LSP server for {language} did not terminate, killing it")
                process.kill()

        del self.servers[language]
        logger.info(f"LSP server for {language} stopped")
        # Close any stderr log handle we opened
        try:
            fh = server.get('stderr_log')
            if fh is not None:
                fh.close()
        except Exception:
            pass
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
                            "didSave": True,
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
            server['initializing'] = False
            
            notification = {
                "jsonrpc": "2.0",
                "method": "initialized",
                "params": {}
            }
            self._send_message(language, notification)

            # Send didChangeConfiguration to trigger LSP to load project settings
            config_notification = {
                "jsonrpc": "2.0",
                "method": "workspace/didChangeConfiguration",
                "params": {
                    "settings": {}
                }
            }
            self._send_message(language, config_notification)

    def _track_request(self, language: str, request_id: int, method: str, file_path: str = None):
        """Track request details for better debugging."""
        if language not in self.request_details:
            self.request_details[language] = {}
        self.request_details[language][request_id] = {
            'method': method,
            'file': file_path or 'workspace',
            'timestamp': time.time()
        }
    
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
    
    def _truncate(self, text: str, limit: int = 200) -> str:
        """Utility: truncate long strings for logging."""
        if text is None:
            return ""
        if len(text) <= limit:
            return text
        return text[:limit] + "..."

    def _log_compact_response(self, what: str, response: Optional[Dict[str, Any]]):
        """Log a compact summary for common LSP responses instead of full payloads."""
        try:
            if not response:
                logger.info(f"{what}: <no response>")
                return
            if 'error' in response and response['error']:
                logger.error(f"{what} error: {self._truncate(json.dumps(response['error']))}")
                return
            result = response.get('result')
            if result is None:
                logger.info(f"{what}: <no result>")
                return

            lw = what.lower()
            if 'completion' in lw:
                if isinstance(result, dict):
                    items = result.get('items') or []
                    logger.info(f"{what}: {len(items)} items")
                elif isinstance(result, list):
                    logger.info(f"{what}: {len(result)} items")
                else:
                    logger.info(f"{what}: <non-standard completion result>")
                return
            if 'references' in lw:
                logger.info(f"{what}: {len(result) if isinstance(result, list) else 0} locations")
                return
            if 'definition' in lw:
                if isinstance(result, list):
                    logger.info(f"{what}: {len(result)} locations")
                elif isinstance(result, dict):
                    logger.info(f"{what}: 1 location")
                else:
                    logger.info(f"{what}: none")
                return
            if 'signature' in lw:
                sigs = result.get('signatures') if isinstance(result, dict) else []
                logger.info(f"{what}: {len(sigs or [])} signatures")
                return
            if 'documentsymbol' in lw:
                names = []
                count = 0
                if isinstance(result, list):
                    count = len(result)
                    for s in result[:5]:
                        if isinstance(s, dict) and 'name' in s:
                            names.append(s['name'])
                logger.info(f"{what}: {count} symbols; sample: {names}")
                return
            if 'workspace/symbol' in lw or 'workspace symbol' in lw or 'workspace' in lw:
                count = 0
                names = []
                if isinstance(result, list):
                    count = len(result)
                    for s in result[:5]:
                        if isinstance(s, dict) and 'name' in s:
                            names.append(s['name'])
                logger.info(f"{what}: {count} symbols; sample: {names}")
                return
            if 'foldingrange' in lw or 'folding' in lw:
                logger.info(f"{what}: {len(result) if isinstance(result, list) else 0} ranges")
                return

            # Default: truncated result
            logger.info(f"{what}: {self._truncate(json.dumps(result))}")
        except Exception as e:
            logger.info(f"{what}: <unable to summarize: {e}>")
    
    def get_definition(self, file_path: str, line: int, character: int, timeout_ms: Optional[int] = None) -> Optional[List[Dict[str, Any]]]:
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
        # Normalize path (expand ~, make absolute) and build proper file URI
        abs_path = Path(file_path).expanduser().resolve()
        file_uri = abs_path.as_uri()
        
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
        
        logger.info(f"Sending textDocument/definition request to {language} LSP for {file_uri} at {line}:{character} (request {request_id})")
        self._track_request(language, request_id, "textDocument/definition", file_path)
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id, timeout=(timeout_ms or 0)/1000.0 or 5.0)
        # Compact log
        self._log_compact_response("textDocument/definition", response)
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
    
    def get_completions(self, file_path: str, line: int, character: int, timeout_ms: Optional[int] = None) -> Optional[Dict[str, Any]]:
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
        # Normalize path (expand ~, make absolute) and build proper file URI
        abs_path = Path(file_path).expanduser().resolve()
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
        
        logger.info(f"Sending textDocument/completion request to {language} LSP for {file_uri} at {line}:{character} (request {request_id})")
        self._track_request(language, request_id, "textDocument/completion", file_path)
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id, timeout=(timeout_ms or 0)/1000.0 or 5.0)
        self._log_compact_response("textDocument/completion", response)
        if response and "result" in response:
            return response["result"]
        elif response and "error" in response:
            logger.error(f"LSP error: {response['error']}")
        return None
    
    def get_references(self, file_path: str, line: int, character: int, include_declaration: bool = True, timeout_ms: Optional[int] = None) -> Optional[List[Dict[str, Any]]]:
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
        # Normalize path (expand ~, make absolute) and build proper file URI
        abs_path = Path(file_path).expanduser().resolve()
        file_uri = abs_path.as_uri()
        
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
        
        logger.info(f"Sending textDocument/references request to {language} LSP for {file_uri} at {line}:{character} (request {request_id})")
        self._track_request(language, request_id, "textDocument/references", file_path)
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id, timeout=(timeout_ms or 0)/1000.0 or 5.0)
        self._log_compact_response("textDocument/references", response)
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
    
    def get_signature_help(self, file_path: str, line: int, character: int, timeout_ms: Optional[int] = None) -> Optional[Dict[str, Any]]:
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
        # Normalize path (expand ~, make absolute) and build proper file URI
        abs_path = Path(file_path).expanduser().resolve()
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
        
        logger.info(f"Sending textDocument/signatureHelp request to {language} LSP for {file_uri} at {line}:{character} (request {request_id})")
        self._track_request(language, request_id, "textDocument/signatureHelp", file_path)
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id, timeout=(timeout_ms or 0)/1000.0 or 5.0)
        self._log_compact_response("textDocument/signatureHelp", response)
        if response and "result" in response:
            return response["result"]
        elif response and "error" in response:
            logger.error(f"LSP error: {response['error']}")
        return None
    
    def get_document_symbols(self, file_path: str, timeout_ms: Optional[int] = None) -> Optional[List[Dict[str, Any]]]:
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
        # Normalize path (expand ~, make absolute) and build proper file URI
        abs_path = Path(file_path).expanduser().resolve()
        file_uri = abs_path.as_uri()
        
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
        
        logger.info(f"Sending textDocument/documentSymbol request to {language} LSP for {file_uri} (request {request_id})")
        self._track_request(language, request_id, "textDocument/documentSymbol", file_path)
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id, timeout=(timeout_ms or 0)/1000.0 or 5.0)
        self._log_compact_response("textDocument/documentSymbol", response)
        
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
    
    def get_workspace_symbols(self, query: str = "", timeout_ms: Optional[int] = None) -> Optional[List[Dict[str, Any]]]:
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
            
            logger.info(f"Sending workspace/symbol request to {language} LSP with query: {query} (request {request_id})")
            self._track_request(language, request_id, "workspace/symbol")
            self._send_message(language, request)
            
            # Wait for the response with our specific request ID
            response = self._wait_for_response(language, request_id, timeout=(timeout_ms or 0)/1000.0 or 5.0)
            self._log_compact_response(f"workspace/symbol [{language}]", response)
            
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
                            response = self._wait_for_response(lang, request_id, timeout=(timeout_ms or 0)/1000.0 or 5.0)
                            
                            if response and "result" in response:
                                result = response["result"]
                                if isinstance(result, list):
                                    for symbol in result:
                                        symbol['language'] = lang
                                    all_symbols.extend(result)
        
        return all_symbols  # Return empty list instead of None
    
    def get_folding_ranges(self, file_path: str, timeout_ms: Optional[int] = None) -> Optional[List[Dict[str, Any]]]:
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
        # Normalize path (expand ~, make absolute) and build proper file URI
        abs_path = Path(file_path).expanduser().resolve()
        file_uri = abs_path.as_uri()
        
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
        
        logger.info(f"Sending textDocument/foldingRange request to {language} LSP for {file_uri} (request {request_id})")
        self._track_request(language, request_id, "textDocument/foldingRange", file_path)
        self._send_message(language, request)
        
        # Wait for the response with our specific request ID
        response = self._wait_for_response(language, request_id, timeout=(timeout_ms or 0)/1000.0 or 5.0)
        self._log_compact_response("textDocument/foldingRange", response)
        
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
            logger.warning(f"Cannot open {file_path} - no {language} server running")
            return
            
        # If server is still initializing, defer the file opening
        server = self.servers.get(language, {})
        if server.get('initializing'):
            # Queue the file to be opened once server is ready
            if not hasattr(self, 'pending_file_opens'):
                self.pending_file_opens = {}
            if language not in self.pending_file_opens:
                self.pending_file_opens[language] = set()
            # Normalize path for consistency
            norm_path = str(Path(file_path).expanduser().resolve())
            self.pending_file_opens[language].add(norm_path)
            logger.info(f"Deferring opening of {norm_path} until {language} server is ready")
            return
            
        # Check if file is already open
        norm_path = str(Path(file_path).expanduser().resolve())
        if norm_path in server.get('open_files', set()):
            logger.debug(f"File {norm_path} is already open in {language} LSP server")
            return
            
        # Read file content
        try:
            with open(norm_path, 'r', encoding='utf-8') as f:
                content = f.read()
        except Exception as e:
            logger.error(f"Failed to read file {norm_path}: {e}")
            return
            
        # Build proper file URI
        file_uri = Path(norm_path).as_uri()
        
        # Detect language ID based on extension
        ext = Path(norm_path).suffix.lower()
        language_id = {
            '.go': 'go',
            '.c': 'c',
            '.h': 'c',
            '.m': 'objective-c',
            '.cpp': 'cpp',
            '.cc': 'cpp',
            '.cxx': 'cpp',
            '.hpp': 'cpp',
            '.hh': 'cpp',
            '.hxx': 'cpp',
            '.py': 'python',
            '.pyw': 'python'
        }.get(ext, 'plaintext')
        
        # Initialize version tracking for this file
        if 'file_versions' not in self.servers[language]:
            self.servers[language]['file_versions'] = {}
        self.servers[language]['file_versions'][norm_path] = 1
        
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
        logger.info(f"Sent textDocument/didOpen for {norm_path}")

        # Track that this file is open (normalized path)
        self.servers[language]['open_files'].add(norm_path)
    
    def _send_did_change(self, language: str, file_path: str, content: str):
        """Send textDocument/didChange notification to the LSP server."""
        if language not in self.servers:
            return
            
        server = self.servers[language]
        # Normalize path for consistency with _open_file
        norm_path = str(Path(file_path).expanduser().resolve())

        # Check if server is initialized and file is open
        if server.get('initializing'):
            logger.debug(f"Skipping didChange for {norm_path} - server still initializing")
            return
        if norm_path not in server.get('open_files', set()):
            logger.warning(f"Skipping didChange for {norm_path} - file not in open_files set: {server.get('open_files', set())}")
            return
            
        # Increment version number for this file
        if 'file_versions' not in server:
            server['file_versions'] = {}
        
        current_version = server['file_versions'].get(norm_path, 0)
        new_version = current_version + 1
        server['file_versions'][norm_path] = new_version
        
        # Build proper file URI
        file_uri = Path(norm_path).as_uri()
        
        notification = {
            "jsonrpc": "2.0",
            "method": "textDocument/didChange",
            "params": {
                "textDocument": {
                    "uri": file_uri,
                    "version": new_version
                },
                "contentChanges": [
                    {
                        "text": content  # Full document sync
                    }
                ]
            }
        }
        
        self._send_message(language, notification)
        logger.debug(f"Sent didChange for {file_path} version {new_version}")

    def _send_did_save(self, language: str, file_path: str, content: str):
        """Send textDocument/didSave notification to the LSP server."""
        if language not in self.servers:
            return

        server = self.servers[language]
        # Normalize path for consistency with _open_file
        norm_path = str(Path(file_path).expanduser().resolve())

        # Check if server is initialized and file is open
        if server.get('initializing') or norm_path not in server.get('open_files', set()):
            return

        # Build proper file URI
        file_uri = Path(norm_path).as_uri()

        notification = {
            "jsonrpc": "2.0",
            "method": "textDocument/didSave",
            "params": {
                "textDocument": {
                    "uri": file_uri
                },
                "text": content  # Include text content for full sync
            }
        }

        self._send_message(language, notification)
        logger.debug(f"Sent didSave for {file_path}")

    def _read_lsp_message(self, language: str) -> Optional[Dict[str, Any]]:
        """Read a single LSP message from the server."""
        if language not in self.servers:
            return None
            
        process = self.servers[language]['process']
        
        try:
            # Read headers line by line until we hit an empty line
            headers = {}
            while True:
                line = process.stdout.readline()
                if not line:  # EOF
                    logger.debug(f"{language} LSP server closed connection")
                    return None
                
                # Check for end of headers (empty line)
                # LSP uses \r\n line endings
                if line == '\r\n' or line == '\n':
                    break
                    
                # Remove trailing whitespace
                line = line.rstrip()
                
                # Parse header: "Key: Value"
                if ':' in line:
                    key, value = line.split(':', 1)
                    headers[key.strip()] = value.strip()
            
            # Content-Length is required
            if 'Content-Length' not in headers:
                logger.error(f"{language} LSP: No Content-Length header found. Headers: {headers}")
                return None
            
            # Read the content based on Content-Length
            content_length = int(headers['Content-Length'])
            content = process.stdout.read(content_length)
            
            if len(content) != content_length:
                logger.error(f"{language} LSP: Expected {content_length} bytes, got {len(content)}")
                return None
            
            # Parse JSON
            return json.loads(content)
                
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse JSON from {language} LSP server: {e}")
            logger.debug(f"Content that failed to parse: {repr(content[:500])}")
        except ValueError as e:
            logger.error(f"Invalid Content-Length from {language} LSP server: {e}")
        except Exception as e:
            logger.error(f"Failed to read message from {language} LSP server: {e}", exc_info=True)
            
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

            # Forward diagnostics to editor as notification
            self._notify_editor_diagnostics(uri, diagnostics)

    def _notify_editor_diagnostics(self, uri: str, raw_diagnostics: list):
        """Send diagnostic notification to the editor."""
        # Convert file:// URI to path
        if uri.startswith('file://'):
            file_path = uri[7:]  # Remove 'file://' prefix
        else:
            file_path = uri

        # Convert LSP diagnostics to our format
        diagnostics_list = []
        for d in raw_diagnostics:
            rng = d.get('range', {})
            start = rng.get('start', {})
            end = rng.get('end', {})
            # LSP lines are 0-based; convert to 1-based
            line_1_based = int(start.get('line', 0)) + 1
            start_char = int(start.get('character', 0) or 0)
            end_char = int(end.get('character', start_char) or start_char)
            severity_num = int(d.get('severity', 0) or 0)
            if severity_num == 1:
                sev = 'error'
            elif severity_num == 2:
                sev = 'warning'
            elif severity_num == 3:
                sev = 'info'
            elif severity_num == 4:
                sev = 'hint'
            else:
                sev = 'info'
            diagnostics_list.append({
                'line': line_1_based,
                'start_char': start_char,
                'end_char': end_char,
                'severity': sev,
                'message': d.get('message', '')
            })

        # Send notification to editor (JSON-RPC notification has no "id" field)
        notification = {
            'jsonrpc': '2.0',
            'method': 'diagnostics',
            'params': {
                'path': file_path,
                'diagnostics': diagnostics_list
            }
        }
        print(json.dumps(notification), flush=True)
        logger.debug(f"Sent diagnostic notification to editor for {file_path} with {len(diagnostics_list)} diagnostics")

    def _notify_editor_lsp_status(self, language: str, status: str, error: str = ""):
        """Send an lsp_status notification to the editor via stdout."""
        extensions = []
        if language in self.language_servers:
            extensions = self.language_servers[language]['extensions']
        notification = {
            'jsonrpc': '2.0',
            'method': 'lsp_status',
            'params': {
                'language': language,
                'status': status,
                'error': error,
                'extensions': extensions,
            }
        }
        print(json.dumps(notification), flush=True)
        logger.debug(f"Sent lsp_status notification: language={language} status={status} error={error}")

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
            # Clean up request details
            if language in self.request_details and request_id in self.request_details[language]:
                del self.request_details[language][request_id]
            return response
        else:
            # Timeout - provide detailed error message
            del self.pending_requests[language][request_id]
            
            # Get request details for better error message
            details_msg = ""
            if language in self.request_details and request_id in self.request_details[language]:
                details = self.request_details[language][request_id]
                details_msg = f" ({language} LSP: {details['method']} for {details.get('file', 'unknown file')})"
                del self.request_details[language][request_id]
            else:
                details_msg = f" ({language} LSP)"
            
            logger.error(f"Timeout waiting for response to request {request_id}{details_msg} after {timeout}s")
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


def log_proxy_method(func):
    """Decorator to log proxy method calls with parameters, results, and timing."""
    def wrapper(*args, **kwargs):
        method_name = func.__name__
        start_time = time.time()
        
        # Format parameters (skip 'self' which is args[0])
        params = []
        if len(args) > 1:
            # Get parameter names from function signature
            import inspect
            sig = inspect.signature(func)
            param_names = list(sig.parameters.keys())[1:]  # Skip 'self'
            for i, (param_name, value) in enumerate(zip(param_names, args[1:])):
                # Truncate long values for logging
                str_value = str(value)
                if len(str_value) > 100:
                    str_value = str_value[:100] + "..."
                params.append(f"{param_name}={str_value}")
        
        # Add keyword arguments
        for key, value in kwargs.items():
            str_value = str(value)
            if len(str_value) > 100:
                str_value = str_value[:100] + "..."
            params.append(f"{key}={str_value}")
        
        param_str = ", ".join(params) if params else "no parameters"
        logger.info(f"[{method_name}] Starting with {param_str}")
        
        try:
            result = func(*args, **kwargs)
            elapsed = time.time() - start_time
            
            # Log result summary - include key context like file path
            context = ""
            # Extract key parameters for context
            if len(args) > 1:
                import inspect
                sig = inspect.signature(func)
                param_names = list(sig.parameters.keys())[1:]  # Skip 'self'
                # Look for key parameters
                for i, (param_name, value) in enumerate(zip(param_names, args[1:])):
                    if param_name in ['path', 'file_path', 'file', 'uri']:
                        context = f" for {value}"
                        break
                    elif param_name == 'language':
                        context = f" ({value} LSP)"
            
            if isinstance(result, dict):
                if 'success' in result:
                    summary = f"success={result['success']}"
                    if 'error' in result:
                        summary += f", error={result['error']}"
                    elif 'result' in result and isinstance(result['result'], list):
                        summary += f", items={len(result['result'])}"
                    elif 'diagnostics' in result:
                        summary += f", diagnostics={len(result['diagnostics'])}"
                elif 'content' in result:
                    summary = f"content_size={len(result.get('content', ''))}"
                elif 'session_id' in result:
                    summary = f"session_id={result['session_id']}"
                else:
                    summary = f"keys={list(result.keys())}"
            elif isinstance(result, list):
                summary = f"{len(result)} items"
            else:
                summary = str(result)[:100]
            
            logger.info(f"[{method_name}] Completed in {elapsed:.3f}s{context}: {summary}")
            return result
            
        except Exception as e:
            elapsed = time.time() - start_time
            logger.error(f"[{method_name}] Failed in {elapsed:.3f}s: {e}")
            raise
    
    return wrapper

class VedaProxy:
    """Main proxy class handling JSON RPC requests for various operations."""

    def __init__(self, test_lsp_overrides=None):
        self.lsp = LSPManager(test_lsp_overrides=test_lsp_overrides)
        self.term = VedaProxy.TerminalManager()
        self.methods = {
            'get_file_attributes': self.get_file_attributes,
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
            'review_pr': self.review_pr,
            'checkout_pr': self.checkout_pr,
            'get_git_file_version': self.get_git_file_version,
            'run_git_blame': self.run_git_blame,
            'get_git_commit_info': self.get_git_commit_info,
            'stage_lines': self.stage_lines,
            'unstage_lines': self.unstage_lines,
            'drop_lines': self.drop_lines,
            'notify_file_change': self.notify_file_change,
            # Terminal methods
            'start_terminal': self.start_terminal,
            'write_terminal': self.write_terminal,
            'read_terminal': self.read_terminal,
            'resize_terminal': self.resize_terminal,
            'close_terminal': self.close_terminal,
            # High quality text diff
            'compute_text_diff': self.compute_text_diff,
        }

    def _run_shell_command(self, cmd, cwd=None, timeout=None, input=None):
        """Run a shell command with comprehensive logging."""
        # Log the command being executed
        cmd_str = ' '.join(cmd) if isinstance(cmd, list) else cmd
        logger.info(f"Running shell command: {cmd_str}")
        if cwd:
            logger.info(f"  Working directory: {cwd}")
        if input:
            logger.info(f"  With input: {len(input)} characters")
        
        try:
            result = subprocess.run(
                cmd, 
                capture_output=True, 
                text=True, 
                cwd=cwd,
                timeout=timeout,
                input=input
            )
            
            # Log stdout information
            stdout_lines = result.stdout.splitlines() if result.stdout else []
            stdout_char_count = len(result.stdout) if result.stdout else 0
            logger.info(f"  Exit code: {result.returncode}")
            logger.info(f"  Stdout: {len(stdout_lines)} lines, {stdout_char_count} characters")
            
            # Log first 5 lines of output
            if stdout_lines:
                preview_lines = stdout_lines[:5]
                for i, line in enumerate(preview_lines, 1):
                    # Truncate long lines to 200 characters
                    if len(line) > 200:
                        line = line[:200] + "..."
                    logger.info(f"    Line {i}: {line}")
                if len(stdout_lines) > 5:
                    logger.info(f"    ... and {len(stdout_lines) - 5} more lines")
            
            # Log stderr if present
            if result.stderr:
                stderr_lines = result.stderr.splitlines()
                logger.warning(f"  Stderr: {len(stderr_lines)} lines, {len(result.stderr)} characters")
                # Log first 5 lines of stderr
                for i, line in enumerate(stderr_lines[:5], 1):
                    if len(line) > 200:
                        line = line[:200] + "..."
                    logger.warning(f"    Stderr line {i}: {line}")
                if len(stderr_lines) > 5:
                    logger.warning(f"    ... and {len(stderr_lines) - 5} more stderr lines")
            
            return result
            
        except subprocess.TimeoutExpired as e:
            logger.error(f"  Command timed out after {timeout} seconds")
            logger.error(f"  Partial stdout: {e.stdout[:500] if e.stdout else 'None'}")
            logger.error(f"  Partial stderr: {e.stderr[:500] if e.stderr else 'None'}")
            raise
        except Exception as e:
            logger.error(f"  Command failed with exception: {e}")
            raise

    @log_proxy_method
    def compute_text_diff(self, left: str, right: str) -> Dict[str, Any]:
        """
        Compute high-quality line diff using Python difflib.SequenceMatcher.
        Returns a list of aligned dicts: {'tag','left','right'}.
        """
        try:
            left_lines = left.split('\n')
            right_lines = right.split('\n')
            if left_lines and left_lines[-1] == '':
                left_lines = left_lines[:-1]
            if right_lines and right_lines[-1] == '':
                right_lines = right_lines[:-1]

            left_keys = [' '.join(line.split()) for line in left_lines]
            right_keys = [' '.join(line.split()) for line in right_lines]
            sm = difflib.SequenceMatcher(a=left_keys, b=right_keys, autojunk=False)
            aligned: List[Dict[str, str]] = []
            for tag, i1, i2, j1, j2 in sm.get_opcodes():
                if tag == 'equal':
                    for ai, bi in zip(range(i1, i2), range(j1, j2)):
                        line_tag = 'equal' if left_lines[ai] == right_lines[bi] else 'replace'
                        aligned.append({'tag': line_tag, 'left': left_lines[ai], 'right': right_lines[bi]})
                elif tag == 'delete':
                    for ai in range(i1, i2):
                        aligned.append({'tag': 'delete', 'left': left_lines[ai], 'right': ''})
                elif tag == 'insert':
                    for bi in range(j1, j2):
                        aligned.append({'tag': 'insert', 'left': '', 'right': right_lines[bi]})
                elif tag == 'replace':
                    # Pair min count as replace; remainder split as delete/insert
                    n = min(i2 - i1, j2 - j1)
                    for k in range(n):
                        aligned.append({'tag': 'replace', 'left': left_lines[i1 + k], 'right': right_lines[j1 + k]})
                    for ai in range(i1 + n, i2):
                        aligned.append({'tag': 'delete', 'left': left_lines[ai], 'right': ''})
                    for bi in range(j1 + n, j2):
                        aligned.append({'tag': 'insert', 'left': '', 'right': right_lines[bi]})

            return {'success': True, 'aligned': aligned}
        except Exception as e:
            logger.error(f"compute_text_diff failed: {e}")
            return {'success': False, 'error': str(e)}

    # ---- Terminal (PTY) support ----
    class _TerminalSession:
        def __init__(self, master_fd: int, pid: int):
            self.master_fd = master_fd
            self.pid = pid
            self.buffer = bytearray()
            self.closed = False
            self.exit_code = None

    class TerminalManager:
        def __init__(self_outer):
            self_outer.sessions = {}
            self_outer.lock = threading.Lock()
            self_outer.sel = selectors.DefaultSelector()

        def start(self_outer, working_dir: str, command: Optional[List[str]], env: Optional[Dict[str,str]], cols: int, rows: int) -> Tuple[str, int]:
            if not command or len(command) == 0:
                shell = os.environ.get('SHELL', '/bin/sh')
                command = [shell]
            # Create PTY
            master_fd, slave_fd = pty.openpty()
            # Set window size
            # Set window size via TIOCSWINSZ
            winsize = struct.pack('HHHH', rows, cols, 0, 0)
            fcntl.ioctl(slave_fd, termios.TIOCSWINSZ, winsize)
            # Prepare env
            env_vars = os.environ.copy()
            env_vars['TERM'] = 'xterm-256color'
            if env:
                env_vars.update(env)
            # Spawn
            pid = os.fork()
            if pid == 0:
                # Child
                os.setsid()
                # Make the slave the controlling terminal for this session
                try:
                    fcntl.ioctl(slave_fd, termios.TIOCSCTTY, 0)
                except Exception:
                    pass
                os.close(master_fd)
                os.dup2(slave_fd, 0)
                os.dup2(slave_fd, 1)
                os.dup2(slave_fd, 2)
                if slave_fd > 2:
                    os.close(slave_fd)
                try:
                    if working_dir:
                        os.chdir(os.path.expanduser(working_dir))
                except Exception:
                    pass
                try:
                    os.execvpe(command[0], command, env_vars)
                except Exception as e:
                    os.write(1, str(e).encode())
                    os._exit(1)
            # Parent
            os.close(slave_fd)
            # Non-blocking master
            fl = fcntl.fcntl(master_fd, fcntl.F_GETFL)
            fcntl.fcntl(master_fd, fcntl.F_SETFL, fl | os.O_NONBLOCK)
            sid = str(uuid.uuid4())
            sess = VedaProxy._TerminalSession(master_fd, pid)
            with self_outer.lock:
                self_outer.sessions[sid] = sess
            # Start reader thread
            def reader():
                while True:
                    try:
                        data = os.read(master_fd, 65536)
                        if not data:
                            break
                        with self_outer.lock:
                            sess.buffer.extend(data)
                    except BlockingIOError:
                        time.sleep(0.01)
                    except OSError:
                        break
                # Wait for child
                _, status = os.waitpid(pid, 0)
                code = os.WEXITSTATUS(status) if os.WIFEXITED(status) else None
                with self_outer.lock:
                    sess.closed = True
                    sess.exit_code = code
            t = threading.Thread(target=reader, daemon=True)
            t.start()
            return sid, pid

        def write(self_outer, session_id: str, data: bytes):
            with self_outer.lock:
                sess = self_outer.sessions.get(session_id)
                if not sess:
                    raise ValueError('invalid session')
                os.write(sess.master_fd, data)

        def read(self_outer, session_id: str, max_bytes: int) -> Tuple[bytes, bool, Optional[int]]:
            with self_outer.lock:
                sess = self_outer.sessions.get(session_id)
                if not sess:
                    raise ValueError('invalid session')
                if max_bytes <= 0:
                    max_bytes = 65536
                n = min(len(sess.buffer), max_bytes)
                if n == 0:
                    return b"", sess.closed, sess.exit_code
                data = bytes(sess.buffer[:n])
                del sess.buffer[:n]
                return data, sess.closed, sess.exit_code

        def resize(self_outer, session_id: str, cols: int, rows: int):
            with self_outer.lock:
                sess = self_outer.sessions.get(session_id)
                if not sess:
                    raise ValueError('invalid session')
                winsize = struct.pack('HHHH', rows, cols, 0, 0)
                fcntl.ioctl(sess.master_fd, termios.TIOCSWINSZ, winsize)

        def close(self_outer, session_id: str):
            with self_outer.lock:
                sess = self_outer.sessions.pop(session_id, None)
                if not sess:
                    return
                try:
                    os.close(sess.master_fd)
                except Exception:
                    pass

    @log_proxy_method
    def start_terminal(self, working_dir: str = '', command: Optional[List[str]] = None, env: Optional[Dict[str,str]] = None, cols: int = 80, rows: int = 24) -> Dict[str, Any]:
        try:
            sid, pid = self.term.start(working_dir, command, env, cols, rows)
            return { 'session_id': sid, 'pid': pid }
        except Exception as e:
            return { 'error': str(e) }

    @log_proxy_method
    def write_terminal(self, session_id: str, data_b64: str) -> Dict[str, Any]:
        try:
            data = base64.b64decode(data_b64)
            self.term.write(session_id, data)
            return { 'success': True }
        except Exception as e:
            return { 'error': str(e) }

    @log_proxy_method
    def read_terminal(self, session_id: str, max_bytes: int = 65536) -> Dict[str, Any]:
        try:
            data, closed, exit_code = self.term.read(session_id, max_bytes)
            return { 'data_b64': base64.b64encode(data).decode(), 'closed': closed, 'exit_code': exit_code }
        except Exception as e:
            return { 'error': str(e) }

    @log_proxy_method
    def resize_terminal(self, session_id: str, cols: int, rows: int) -> Dict[str, Any]:
        try:
            self.term.resize(session_id, cols, rows)
            return { 'success': True }
        except Exception as e:
            return { 'error': str(e) }

    @log_proxy_method
    def close_terminal(self, session_id: str) -> Dict[str, Any]:
        try:
            self.term.close(session_id)
            return { 'success': True }
        except Exception as e:
            return { 'error': str(e) }

    @log_proxy_method
    def get_file_attributes(self, path: str) -> Dict[str, Any]:
        """
        Get attributes of a file or directory.
        
        Args:
            path: File or directory path
            
        Returns:
            Dictionary with file/directory attributes
        """
        file_path = Path(path).expanduser().resolve()
        
        if not file_path.exists():
            raise FileNotFoundError(f"Path does not exist: {path}")
        
        stat_info = file_path.stat()
        
        # Determine file type
        if file_path.is_dir():
            file_type = 'directory'
        elif file_path.is_file():
            file_type = 'file'
        elif file_path.is_symlink():
            file_type = 'symlink'
        else:
            file_type = 'unknown'
        
        attributes = {
            'name': file_path.name,
            'path': str(file_path),
            'type': file_type,
            'size': stat_info.st_size,
            'mode': oct(stat_info.st_mode),
            'modified': stat_info.st_mtime,
            'created': getattr(stat_info, 'st_birthtime', stat_info.st_ctime),
            'accessed': stat_info.st_atime,
            'is_directory': file_path.is_dir(),
            'is_file': file_path.is_file(),
            'is_symlink': file_path.is_symlink(),
            'exists': True
        }
        
        # Add permissions info
        attributes['readable'] = os.access(str(file_path), os.R_OK)
        attributes['writable'] = os.access(str(file_path), os.W_OK)
        attributes['executable'] = os.access(str(file_path), os.X_OK)
        
        return attributes

    @log_proxy_method
    def list_directory_recursive(self, path: str, filter_mode: str = "supported") -> List[Dict[str, Any]]:
        """
        Recursively list all files and directories in the specified path and its subdirectories.
        
        Args:
            path: Directory path to start from
            filter_mode: "supported" to list only supported files, "all" to list all files
            
        Returns:
            List of file/directory information dictionaries with absolute paths
        """
        # Define supported file extensions for programming languages we can edit
        supported_extensions = {
            # Python
            '.py', '.pyw', '.pyi',
            # Go
            '.go', '.mod', '.sum',
            # C/C++/Objective-C
            '.c', '.h', '.m', '.cpp', '.cc', '.cxx', '.hpp', '.hh', '.hxx', '.C', '.H',
            # Shell scripts
            '.sh', '.bash', '.zsh', '.fish', '.ksh', '.csh', '.tcsh',
            # Makefiles and build files
            '.mk', '.make',
            # Common text formats and templates
            '.txt', '.md', '.markdown', '.rst', '.yaml', '.yml', '.json', '.toml', '.ini', '.cfg', '.conf',
            '.j2',  # Jinja2 templates
            # Web files
            '.html', '.htm', '.css', '.js', '.jsx', '.ts', '.tsx',
            # LaTeX/TeX
            '.tex', '.latex', '.sty', '.cls', '.dtx', '.ltx', '.bib', '.bbl', '.bst',
            # Other programming languages (for future LSP support)
            '.rs', '.java', '.rb', '.php', '.swift', '.kt', '.scala', '.clj',
            # Configuration and documentation
            '.xml', '.gitignore', '.dockerignore', '.env', '.properties',
        }
        
        # Also include files without extensions that are commonly editable
        supported_filenames = {
            'Makefile', 'makefile', 'GNUmakefile', 'Dockerfile', 'docker-compose.yml',
            'README', 'LICENSE', 'CHANGELOG', 'AUTHORS', 'CONTRIBUTORS',
            '.bashrc', '.zshrc', '.vimrc', '.gitconfig', '.profile'
        }
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
                            
                        # Apply filter for files (always include directories)
                        if filter_mode == "supported" and entry.is_file():
                            # Check if file extension or name is supported
                            ext = entry.suffix.lower()
                            name = entry.name
                            if ext and ext not in supported_extensions and name not in supported_filenames:
                                continue
                            
                        # Create simplified file info for fuzzy listing
                        stat_info = entry.stat()
                        file_info = {
                            'name': entry.name,
                            'path': str(entry.absolute()),
                            'type': 'directory' if entry.is_dir() else 'file',
                            'size': stat_info.st_size if entry.is_file() else 0,
                            'mtime': stat_info.st_mtime
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
        
        # Sort by modification time (newest first)
        entries.sort(key=lambda x: x['mtime'], reverse=True)
        
        return entries

    @log_proxy_method
    def get_file(self, path: str) -> Dict[str, str]:
        """
        Read the contents of a file.
        
        Args:
            path: Absolute file path to read
            
        Returns:
            Dictionary with 'content' key containing the file contents
        """
        start_time = time.time()  # Initialize start_time for performance tracking
        file_path = Path(path).expanduser().resolve()
        
        if not file_path.exists():
            # For non-existent files, return empty content to allow creating new files
            logger.info(f"File does not exist, returning empty content for new file: {path}")
            return {
                'content': '',
                'changes': {'added': [], 'deleted': [], 'modified': []},
                'branch': 'main',
                'diagnostics': [],
                'lsp_error': '',
            }
            
        if not file_path.is_file():
            raise ValueError(f"Path is not a file: {path}")
            
        try:
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

                # Do not block here; diagnostics will be collected from cache below when available
            
            lsp_time = time.time()
            logger.info(f"LSP operations took {(lsp_time - read_time) * 1000:.1f}ms")
            
            # Get current git branch
            branch = "main"  # default
            try:
                cmd_branch = ['git', 'rev-parse', '--abbrev-ref', 'HEAD']
                result_branch = self._run_shell_command(cmd_branch, cwd=file_path.parent)
                if result_branch.returncode == 0:
                    branch = result_branch.stdout.strip()
            except Exception as e:
                logger.warning(f"Could not get git branch: {e}")
            
            branch_time = time.time()
            logger.info(f"Git branch detection took {(branch_time - lsp_time) * 1000:.1f}ms")
            
            # Get git changes for the file
            changes = self._get_file_git_changes(file_path)
            
            git_time = time.time()
            logger.info(f"Git diff operations took {(git_time - branch_time) * 1000:.1f}ms")
            logger.info(f"Total get_file time: {(git_time - start_time) * 1000:.1f}ms")
            
            # Collect diagnostics for this file from LSP cache (after short warm-up)
            diagnostics_list = []
            try:
                uri = f"file://{str(file_path)}"
                raw_diags = self.lsp.diagnostics.get(uri, []) or []
                for d in raw_diags:
                    rng = d.get('range', {})
                    start = rng.get('start', {})
                    end = rng.get('end', {})
                    # LSP lines are 0-based; convert to 1-based
                    line_1_based = int(start.get('line', 0)) + 1
                    start_char = int(start.get('character', 0) or 0)
                    end_char = int(end.get('character', start_char) or start_char)
                    severity_num = int(d.get('severity', 0) or 0)
                    if severity_num == 1:
                        sev = 'error'
                    elif severity_num == 2:
                        sev = 'warning'
                    elif severity_num == 3:
                        sev = 'info'
                    elif severity_num == 4:
                        sev = 'hint'
                    else:
                        sev = 'info'
                    diagnostics_list.append({
                        'line': line_1_based,
                        'severity': sev,
                        'message': d.get('message', ''),
                        'start_char': start_char,
                        'end_char': end_char,
                    })
            except Exception as e:
                logger.warning(f"Failed to translate diagnostics for {path}: {e}")

            return {
                'content': content,
                'changes': changes,
                'branch': branch,
                'diagnostics': diagnostics_list,
                'lsp_error': '',
            }
        except UnicodeDecodeError:
            # Try reading as binary and return error
            raise ValueError(f"File is not a text file: {path}")
        except PermissionError:
            raise PermissionError(f"Permission denied: {path}")

    @log_proxy_method
    def write_file(self, path: str, content: str) -> Dict[str, Any]:
        """
        Write content to a file and return diagnostics and git changes.
        
        Args:
            path: Absolute file path to write
            content: Content to write to the file
            
        Returns:
            Dictionary with 'success' key, 'diagnostics' list, and 'changes' dict
        """
        file_path = Path(path).expanduser().resolve()
        
        try:
            # Create parent directories if they don't exist
            file_path.parent.mkdir(parents=True, exist_ok=True)
            
            # Preserve original file permissions if file exists
            original_mode = None
            if file_path.exists():
                try:
                    original_mode = file_path.stat().st_mode
                except OSError:
                    # If we can't stat the file, continue without preserving permissions
                    pass
            
            # Write to a temporary file first for atomic operation
            # Create temp file in same directory with random suffix
            import random
            import string
            random_suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
            temp_path = file_path.parent / f"{file_path.name}.{random_suffix}.tmp"
            
            try:
                # Write content to temp file
                temp_path.write_text(content, encoding='utf-8')
                
                # If we have original permissions, apply them to the temp file
                if original_mode is not None:
                    try:
                        os.chmod(temp_path, original_mode)
                    except OSError:
                        # If we can't set permissions, continue anyway
                        pass
                
                # Atomically replace the original file
                # On POSIX systems, this is atomic
                temp_path.replace(file_path)
            except:
                # Clean up temp file if something goes wrong
                try:
                    temp_path.unlink()
                except:
                    pass
                raise
            
            # Notify LSP server about the file change
            language = self.lsp.get_language_for_file(str(file_path))
            if language and language in self.lsp.servers:
                # Send didChange first to update the content, then didSave
                self.lsp._send_did_change(language, str(file_path), content)
                self.lsp._send_did_save(language, str(file_path), content)
                logger.debug(f"Sent didChange and didSave notifications to LSP server for {language}")
            
            # Get git changes for the file after save
            changes = self._get_file_git_changes(file_path)
            
            # Collect diagnostics for this file from LSP cache
            diagnostics_list = []
            try:
                uri = f"file://{str(file_path)}"
                raw_diags = self.lsp.diagnostics.get(uri, []) or []
                for d in raw_diags:
                    rng = d.get('range', {})
                    start = rng.get('start', {})
                    end = rng.get('end', {})
                    # LSP lines are 0-based; convert to 1-based
                    line_1_based = int(start.get('line', 0)) + 1
                    start_char = int(start.get('character', 0) or 0)
                    end_char = int(end.get('character', start_char) or start_char)
                    severity_num = int(d.get('severity', 0) or 0)
                    if severity_num == 1:
                        sev = 'error'
                    elif severity_num == 2:
                        sev = 'warning'
                    elif severity_num == 3:
                        sev = 'info'
                    elif severity_num == 4:
                        sev = 'hint'
                    else:
                        sev = 'info'
                    diagnostics_list.append({
                        'line': line_1_based,
                        'severity': sev,
                        'message': d.get('message', ''),
                        'start_char': start_char,
                        'end_char': end_char,
                    })
            except Exception as e:
                logger.warning(f"Failed to translate diagnostics for {path}: {e}")
            
            return {'success': True, 'diagnostics': diagnostics_list, 'changes': changes}
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

        # Handle symlinks (compatible with older Python versions without Path.readlink)
        symlink_target = None
        if path.is_symlink():
            try:
                # Prefer pathlib's readlink when available (Py>=3.9), otherwise fallback to os.readlink
                if hasattr(path, 'readlink'):
                    symlink_target = str(path.readlink())  # type: ignore[attr-defined]
                else:
                    symlink_target = os.readlink(str(path))
            except Exception:
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
    
    @log_proxy_method
    def start_lsp_server(self, language: str) -> Dict[str, Any]:
        """Start the LSP server for the specified language."""
        success = self.lsp.start_server(language)
        return {'success': success}
    
    @log_proxy_method
    def stop_lsp_server(self, language: str) -> Dict[str, Any]:
        """Stop the LSP server for the specified language."""
        success = self.lsp.stop_server(language)
        return {'success': success}
    
    @log_proxy_method
    def go_to_definition(self, file_path: str, line: int, character: int, timeout_ms: Optional[int] = None) -> Dict[str, Any]:
        """Get the definition location for a symbol at the given position."""
        try:
            result = self.lsp.get_definition(file_path, line, character, timeout_ms=timeout_ms)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No definition found'}
        except Exception as e:
            logger.error(f"Error in go_to_definition: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def get_lsp_messages(self, language: str) -> Dict[str, Any]:
        """Get all LSP messages for a specific language."""
        messages = self.lsp.get_message_buffer(language)
        return {'success': True, 'messages': messages}
    
    @log_proxy_method
    def get_lsp_notifications(self, language: str) -> Dict[str, Any]:
        """Get pending showMessage notifications for a language."""
        notifications = self.lsp.get_pending_notifications(language)
        return {'success': True, 'notifications': notifications}
    
    @log_proxy_method
    def get_document_symbols(self, file_path: str, timeout_ms: Optional[int] = None) -> Dict[str, Any]:
        """Get document symbols (functions, classes, etc.) for a file."""
        try:
            result = self.lsp.get_document_symbols(file_path, timeout_ms=timeout_ms)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No symbols found'}
        except Exception as e:
            logger.error(f"Error in get_document_symbols: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def get_folding_ranges(self, file_path: str, timeout_ms: Optional[int] = None) -> Dict[str, Any]:
        """Get folding ranges for a file."""
        try:
            result = self.lsp.get_folding_ranges(file_path, timeout_ms=timeout_ms)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No folding ranges found'}
        except Exception as e:
            logger.error(f"Error in get_folding_ranges: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def get_signature_help(self, file_path: str, line: int, character: int, timeout_ms: Optional[int] = None) -> Dict[str, Any]:
        """Get signature help for a function at the given position."""
        try:
            result = self.lsp.get_signature_help(file_path, line, character, timeout_ms=timeout_ms)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No signature help available'}
        except Exception as e:
            logger.error(f"Error in get_signature_help: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def list_project_symbols(self, query: str = "", timeout_ms: Optional[int] = None) -> Dict[str, Any]:
        """Get all symbols across the entire project workspace."""
        try:
            result = self.lsp.get_workspace_symbols(query, timeout_ms=timeout_ms)
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
            # Return empty list instead of error when no symbols found
            return {'success': True, 'symbols': []}
        except Exception as e:
            logger.error(f"Error in list_project_symbols: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def find_references(self, file_path: str, line: int, character: int, include_declaration: bool = True, timeout_ms: Optional[int] = None) -> Dict[str, Any]:
        """Find all references to a symbol at the given position."""
        try:
            result = self.lsp.get_references(file_path, line, character, include_declaration, timeout_ms=timeout_ms)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No references found'}
        except Exception as e:
            logger.error(f"Error in find_references: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def get_completions(self, file_path: str, line: int, character: int, timeout_ms: Optional[int] = None) -> Dict[str, Any]:
        """Get completion suggestions at the given position."""
        try:
            result = self.lsp.get_completions(file_path, line, character, timeout_ms=timeout_ms)
            if result is not None:
                return {'success': True, 'result': result}
            return {'success': False, 'error': 'No completions found'}
        except Exception as e:
            logger.error(f"Error in get_completions: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def get_diagnostics(self) -> Dict[str, Any]:
        """Get all diagnostics from all open files."""
        try:
            diagnostics = self.lsp.get_all_diagnostics()
            return {'success': True, 'diagnostics': diagnostics}
        except Exception as e:
            logger.error(f"Error in get_diagnostics: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def notify_file_change(self, file_path: str, content: str) -> Dict[str, Any]:
        """
        Notify LSP server about file content changes.
        
        Args:
            file_path: The absolute path to the file
            content: The full updated content of the file
            
        Returns:
            Dictionary with status of the notification
        """
        try:
            # Determine language from file extension
            language = self.lsp.get_language_for_file(file_path)
            
            if not language:
                return {"status": "skipped", "reason": "No LSP server for file type"}
            
            # Check if LSP server is running for this language
            if language not in self.lsp.servers:
                return {"status": "skipped", "reason": f"LSP server not running for {language}"}
            
            # Send the didChange notification
            self.lsp._send_did_change(language, file_path, content)
            
            return {"status": "success", "language": language}
            
        except Exception as e:
            logger.error(f"Failed to notify file change: {e}")
            return {"error": str(e)}
    
    @log_proxy_method
    def checkout_pr(self, working_dir: str, reference: str, head: str, open_files: List[str]) -> Dict[str, Any]:
        """Checkout in place, including pinned nested submodules; never stash or force."""
        result = {'success': False, 'root': '', 'files': []}
        deadline = time.monotonic() + 240

        def run(*args):
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise ValueError('PR checkout timed out')
            proc = self._run_shell_command(list(args), cwd=working_dir, timeout=min(60, remaining))
            if proc.returncode != 0:
                raise ValueError(proc.stderr.strip() or ' '.join(args[:2]) + ' failed')
            return proc.stdout.strip()

        try:
            if not re.fullmatch(r'[0-9a-f]{40}', head):
                raise ValueError('Invalid PR commit ID')
            parsed = urlparse(reference)
            if parsed.scheme != 'https' or not re.fullmatch(r'/[^/]+/[^/]+/pull/[1-9][0-9]*', parsed.path):
                raise ValueError('Invalid PR URL')
            root = Path(run('git', 'rev-parse', '--show-toplevel')).resolve()
            working_dir = str(root)
            repo_url = json.loads(run('gh', 'repo', 'view', '--json', 'url'))['url']
            if repo_url.rstrip('/').lower() != reference.rsplit('/pull/', 1)[0].lower():
                raise ValueError('The PR belongs to a different repository; open its project first')
            # Respect the project's submodule ignore settings for build artifacts.
            # Non-forced checkout/update still reject conflicting local changes.
            if run('git', 'status', '--porcelain', '--untracked-files=normal'):
                raise ValueError('Commit or move local changes before PR checkout (including submodules)')

            # From this point, even a failure may have changed files. Return reload
            # data on failure as well; never claim the previous checkout is intact.
            result['root'] = str(root)
            self.lsp.stop_all_servers()
            self.lsp.pending_file_opens.clear()
            self.lsp.file_versions.clear()
            self.lsp.diagnostics.clear()
            self.lsp.lsp_errors.clear()
            run('gh', 'pr', 'checkout', reference)
            if run('git', 'rev-parse', 'HEAD') != head:
                raise ValueError('PR changed during checkout; reopen it to load the new revision')
            run('git', 'submodule', 'sync', '--recursive')
            run('git', 'submodule', 'update', '--init', '--recursive', '--checkout')
            if run('git', 'status', '--porcelain', '--untracked-files=normal'):
                raise ValueError('Checkout contains local changes; resolve them before reviewing')
            result['success'] = True
        except Exception as e:
            result['error'] = str(e)
            if result['root']:
                result['error'] += ' (checkout may have changed; open buffers have been refreshed)'
        finally:
            if result['root']:
                for name in open_files:
                    path = Path(name).expanduser().absolute()
                    if not path.is_relative_to(root):
                        continue
                    entry = {'path': name, 'content': '', 'error': ''}
                    try:
                        entry['content'] = path.read_text(encoding='utf-8')
                    except Exception as e:
                        entry['error'] = str(e)
                    result['files'].append(entry)
        return result

    @log_proxy_method
    def review_pr(self, working_dir: str, query: Dict[str, Any]) -> Dict[str, Any]:
        """Read PR metadata and immutable file snapshots using the user's gh login."""
        try:
            def gh(*args):
                result = self._run_shell_command(['gh', *args], cwd=working_dir, timeout=60)
                if result.returncode != 0:
                    raise ValueError(result.stderr.strip() or 'GitHub CLI failed')
                return json.loads(result.stdout)

            action = query['action']
            if action == 'list':
                fields = 'number,title,url,author'
                requested = gh('pr', 'list', '--state', 'open', '--limit', '1000',
                               '--search', 'review-requested:@me', '--json', fields)
                assigned = gh('pr', 'list', '--state', 'open', '--limit', '1000',
                              '--assignee', '@me', '--json', fields)
                return {'success': True, 'requested': requested, 'assigned': assigned}

            reference = query.get('reference') or query.get('url', '')
            if re.fullmatch(r'[1-9][0-9]*', reference):
                repo_url = gh('repo', 'view', '--json', 'url')['url']
                reference = repo_url.rstrip('/') + '/pull/' + reference
            parsed = urlparse(reference)
            match = re.fullmatch(r'/([^/]+)/([^/]+)/pull/([1-9][0-9]*)/?', parsed.path)
            if parsed.scheme != 'https' or not parsed.hostname or not match or parsed.username or parsed.port:
                raise ValueError('Enter a positive PR number or an HTTPS PR URL')
            owner, repo, number = match.groups()
            endpoint = 'repos/' + quote(owner, safe='') + '/' + quote(repo, safe='')

            def api(path):
                return gh('api', '--hostname', parsed.hostname, endpoint + path)

            if action == 'view':
                pr = api('/pulls/' + number)
                base, head = pr['base']['sha'], pr['head']['sha']
                comparison = api('/compare/' + base + '...' + head + '?per_page=1')
                files = []
                for page in range(1, 31):
                    batch = api('/pulls/' + number + '/files?per_page=100&page=' + str(page))
                    files.extend(batch)
                    if len(batch) < 100:
                        break
                latest = api('/pulls/' + number)
                if latest['head']['sha'] != head or latest['base']['sha'] != base:
                    raise ValueError('PR changed while loading; open it again to refresh')
                if len(files) != pr['changed_files']:
                    raise ValueError('GitHub did not return the complete file list (maximum 3000 files)')
                fields = ('filename', 'previous_filename', 'status', 'additions', 'deletions')
                files = [{k: f[k] for k in fields if k in f} for f in files]
                return {'success': True,
                        'pr': {'number': pr['number'], 'title': pr['title'], 'body': pr['body'] or '',
                               'url': pr['html_url'], 'author': pr['user']},
                        'base': comparison['merge_base_commit']['sha'], 'head': head, 'files': files}

            if action == 'file':
                base, head = query['base'], query['head']
                if not all(re.fullmatch(r'[0-9a-f]{40}', sha) for sha in (base, head)):
                    raise ValueError('Invalid PR commit ID')

                root_result = self._run_shell_command(['git', 'rev-parse', '--show-toplevel'], cwd=working_dir, timeout=60)
                head_result = self._run_shell_command(['git', 'rev-parse', 'HEAD'], cwd=working_dir, timeout=60)
                if root_result.returncode or head_result.returncode or head_result.stdout.strip() != head:
                    raise ValueError('The checkout no longer matches this PR; reopen the PR before navigating')
                root = Path(root_result.stdout.strip()).resolve()
                working_file = (root / query['path']).resolve()
                if not working_file.is_relative_to(root):
                    raise ValueError('PR file is outside the project')

                def content(path, sha):
                    entry = api('/contents/' + quote(path, safe='/') + '?ref=' + sha)
                    if isinstance(entry, dict) and (entry.get('type') == 'submodule' or entry.get('submodule_git_url')):
                        return ('Subproject commit ' + entry['sha'] + '\n').encode('utf-8')
                    if not isinstance(entry, dict) or entry.get('type') != 'file':
                        raise ValueError('This entry is not a regular file')
                    if entry.get('encoding') != 'base64':
                        raise ValueError('GitHub cannot return this file as text (it may be too large)')
                    return base64.b64decode(entry['content'])

                status = query['status']
                left = b'' if status == 'added' else content(query.get('old_path') or query['path'], base)
                right = b'' if status == 'removed' else content(query['path'], head)
                navigation_path = ''
                if status != 'removed' and working_file.is_file():
                    if working_file.read_bytes() != right:
                        raise ValueError('Working file differs from the PR snapshot; save or resolve local changes first')
                    navigation_path = str(working_file)
                try:
                    if b'\x00' in left or b'\x00' in right:
                        raise UnicodeError()
                    return {'success': True, 'left': left.decode('utf-8'), 'right': right.decode('utf-8'), 'navigation_path': navigation_path}
                except UnicodeError:
                    return {'success': True, 'binary': True}
            raise ValueError('Unknown PR review action')
        except FileNotFoundError:
            return {'success': False, 'error': 'Install gh and run gh auth login on the project host'}
        except Exception as e:
            return {'success': False, 'error': str(e)}

    @log_proxy_method
    def get_project_changes(self, working_dir: str = None) -> Dict[str, Any]:
        """
        Get all changed files in the project with their diff information.
        """
        try:
            # Use provided working directory or current directory
            cwd = Path(working_dir).expanduser().resolve() if working_dir else Path.cwd()

            # Get git root directory - git reports paths relative to this
            git_root_cmd = ['git', 'rev-parse', '--show-toplevel']
            git_root_result = self._run_shell_command(git_root_cmd, cwd=cwd)
            if git_root_result.returncode != 0:
                return {'success': False, 'error': 'Not a git repository'}
            git_root = Path(git_root_result.stdout.strip())

            def to_absolute_path(path_str: str) -> str:
                """Convert git-reported paths to absolute paths anchored at git root."""
                raw_path = Path(path_str).expanduser()
                if raw_path.is_absolute():
                    candidate = raw_path
                else:
                    # Git reports paths relative to git root, not cwd
                    candidate = git_root / raw_path
                try:
                    return str(candidate.resolve(strict=False))
                except TypeError:
                    return str(candidate.resolve())

            # Get list of changed files
            cmd = ['git', 'diff', '--name-status', 'HEAD']
            result = self._run_shell_command(cmd, cwd=cwd)
            
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
            result_staged = self._run_shell_command(cmd_staged, cwd=cwd)
            
            staged_files = {}
            if result_staged.returncode == 0:
                for line in result_staged.stdout.strip().split('\n'):
                    if line:
                        parts = line.split('\t')
                        if len(parts) >= 2:
                            status, filepath = parts[0], parts[1]
                            staged_files[filepath] = status
            
            # Get untracked files (not staged, not in HEAD)
            cmd_untracked = ['git', 'ls-files', '--others', '--exclude-standard']
            result_untracked = self._run_shell_command(cmd_untracked, cwd=cwd)

            untracked_files = []
            if result_untracked.returncode == 0 and result_untracked.stdout:
                for line in result_untracked.stdout.strip().split('\n'):
                    if line:
                        untracked_files.append(line)

            # Combine all changes
            all_files = {}
            for f, status in unstaged_files.items():
                all_files[f] = {'status': status, 'staged': False}
            for f, status in staged_files.items():
                if f in all_files:
                    all_files[f]['staged'] = True
                else:
                    all_files[f] = {'status': status, 'staged': True}

            # Add untracked files as added (A) and unstaged
            for f in untracked_files:
                if f not in all_files:
                    all_files[f] = {'status': 'A', 'staged': False}
            
            # Get detailed changes for each file
            changes = []
            for filepath, info in all_files.items():
                try:
                    # Get the diff for this file
                    if info['staged']:
                        cmd = ['git', 'diff', '--cached', '--', filepath]
                    else:
                        cmd = ['git', 'diff', 'HEAD', '--', filepath]
                    
                    result = self._run_shell_command(cmd, cwd=cwd)
                    if result.returncode == 0:
                        diff_info = self._parse_git_diff(result.stdout)
                        changes.append({
                            'file': to_absolute_path(filepath),
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
    
    def get_git_file_version(self, path: str, revision: str = "HEAD") -> Dict[str, Any]:
        """
        Get a specific git version of a file.
        """
        logger.info(f"get_git_file_version called: path={path}, revision={revision}")
        try:
            # Resolve the file path
            file_path = Path(path).expanduser().resolve()
            logger.info(f"Resolved file path: {file_path}")
            
            # Get the git root directory
            logger.info(f"Running git rev-parse from {file_path.parent}")
            result = subprocess.run(['git', 'rev-parse', '--show-toplevel'], 
                                 capture_output=True, text=True, cwd=file_path.parent,
                                 timeout=5)
            if result.returncode != 0:
                logger.error(f"git rev-parse failed: {result.stderr}")
                return {'success': False, 'error': 'Not a git repository'}
            
            git_root = Path(result.stdout.strip())
            logger.info(f"Git root: {git_root}")
            
            # Get relative path from git root
            try:
                relative_path = file_path.relative_to(git_root)
            except ValueError:
                return {'success': False, 'error': 'File is not in git repository'}
            
            # Get file content at specified revision
            cmd = ['git', 'show', f"{revision}:{relative_path}"]
            logger.info(f"Running git show command: {cmd} in {git_root}")
            result = subprocess.run(cmd, capture_output=True, text=True, cwd=git_root, timeout=10)
            logger.info(f"git show completed with return code: {result.returncode}")
            
            if result.returncode != 0:
                error_msg = result.stderr.strip()
                if "does not exist" in error_msg or "Path" in error_msg:
                    return {'success': False, 'error': f'File does not exist at revision {revision}'}
                elif "bad revision" in error_msg.lower():
                    return {'success': False, 'error': f'Invalid revision: {revision}'}
                else:
                    return {'success': False, 'error': f'Git error: {error_msg}'}
            
            return {
                'success': True,
                'content': result.stdout
            }
            
        except subprocess.TimeoutExpired:
            logger.error(f"Git command timed out for file: {path}")
            return {'success': False, 'error': 'Git command timed out'}
        except Exception as e:
            logger.error(f"Error in get_git_file_version: {e}")
            return {'success': False, 'error': str(e)}
    
    def run_git_blame(self, path: str, line_number: int) -> Dict[str, Any]:
        """
        Run git blame for a specific line of a file.
        """
        logger.info(f"run_git_blame called: path={path}, line_number={line_number}")
        try:
            # Resolve the file path
            file_path = Path(path).expanduser().resolve()
            logger.info(f"Resolved file path: {file_path}")
            
            # Get the git root directory
            result = subprocess.run(['git', 'rev-parse', '--show-toplevel'], 
                                 capture_output=True, text=True, cwd=file_path.parent,
                                 timeout=5)
            
            if result.returncode != 0:
                logger.error(f"Not a git repository: {result.stderr}")
                return {'success': False, 'error': 'Not a git repository'}
            
            git_root = Path(result.stdout.strip())
            logger.info(f"Git root: {git_root}")
            
            # Run git blame for the specific line
            cmd = ['git', 'blame', '-L', f'{line_number},{line_number}', '--porcelain', str(file_path)]
            logger.info(f"Running git blame command: {cmd} in {git_root}")
            result = subprocess.run(cmd, capture_output=True, text=True, cwd=git_root, timeout=10)
            logger.info(f"git blame completed with return code: {result.returncode}")
            
            if result.returncode != 0:
                logger.error(f"git blame failed: {result.stderr}")
                return {'success': False, 'error': result.stderr.strip()}
            
            # Parse the porcelain output
            lines = result.stdout.strip().split('\n')
            if not lines:
                return {'success': False, 'error': 'No blame information available'}
            
            commit_hash = ""
            author = ""
            author_time = ""
            summary = ""
            
            for line in lines:
                if line.startswith('author '):
                    author = line[7:]
                elif line.startswith('author-time '):
                    # Convert Unix timestamp to readable date
                    timestamp = int(line[12:])
                    from datetime import datetime
                    author_time = datetime.fromtimestamp(timestamp).strftime('%Y-%m-%d %H:%M:%S')
                elif line.startswith('summary '):
                    summary = line[8:]
                elif not commit_hash and line and not line.startswith('\t'):
                    # First line contains: <hash> <original-line> <final-line> <num-lines>
                    parts = line.split()
                    if parts and len(parts[0]) == 40 and all(c in '0123456789abcdef' for c in parts[0]):
                        commit_hash = parts[0]
            
            logger.info(f"Parsed blame: commit={commit_hash[:7] if commit_hash else 'none'}, author={author}, date={author_time}, message={summary}")
            
            return {
                'success': True,
                'commit': commit_hash,
                'author': author,
                'date': author_time,
                'message': summary
            }
            
        except subprocess.TimeoutExpired:
            logger.error(f"Git blame command timed out for file: {path}")
            return {'success': False, 'error': 'Git command timed out'}
        except Exception as e:
            logger.error(f"Error in run_git_blame: {e}")
            return {'success': False, 'error': str(e)}
    
    def get_git_commit_info(self, path: str, line_number: int) -> Dict[str, Any]:
        """
        Get full commit information for a specific line of a file.
        First runs git blame to get the commit hash, then git show to get full details.
        """
        logger.info(f"get_git_commit_info called: path={path}, line_number={line_number}")
        try:
            # First get the blame info to find the commit hash
            blame_result = self.run_git_blame(path, line_number)
            if not blame_result.get('success'):
                return blame_result
            
            commit_hash = blame_result.get('commit')
            if not commit_hash:
                return {'success': False, 'error': 'No commit hash found for this line'}
            
            # Resolve the file path
            file_path = Path(path).expanduser().resolve()
            
            # Get the git root directory
            result = subprocess.run(['git', 'rev-parse', '--show-toplevel'], 
                                 capture_output=True, text=True, cwd=file_path.parent,
                                 timeout=5)
            
            if result.returncode != 0:
                logger.error(f"Not a git repository: {result.stderr}")
                return {'success': False, 'error': 'Not a git repository'}
            
            git_root = Path(result.stdout.strip())
            
            # Run git show to get full commit details with diff
            cmd = ['git', 'show', '--no-color', commit_hash]
            logger.info(f"Running git show command: {cmd} in {git_root}")
            result = subprocess.run(cmd, capture_output=True, text=True, cwd=git_root, timeout=10)
            
            if result.returncode != 0:
                logger.error(f"git show failed: {result.stderr}")
                return {'success': False, 'error': result.stderr.strip()}
            
            # Also get commit info in a structured way
            cmd_info = ['git', 'show', '--no-patch', '--format=%H%n%an%n%ae%n%at%n%s%n%b', commit_hash]
            info_result = subprocess.run(cmd_info, capture_output=True, text=True, cwd=git_root, timeout=5)
            
            info_lines = info_result.stdout.strip().split('\n')
            commit_info = {
                'hash': info_lines[0] if len(info_lines) > 0 else commit_hash,
                'author': info_lines[1] if len(info_lines) > 1 else '',
                'email': info_lines[2] if len(info_lines) > 2 else '',
                'timestamp': info_lines[3] if len(info_lines) > 3 else '',
                'subject': info_lines[4] if len(info_lines) > 4 else '',
                'body': '\n'.join(info_lines[5:]) if len(info_lines) > 5 else ''
            }
            
            # Convert timestamp to readable date
            if commit_info['timestamp']:
                try:
                    from datetime import datetime
                    timestamp = int(commit_info['timestamp'])
                    commit_info['date'] = datetime.fromtimestamp(timestamp).strftime('%Y-%m-%d %H:%M:%S')
                except:
                    commit_info['date'] = commit_info['timestamp']
            else:
                commit_info['date'] = ''
            
            return {
                'success': True,
                'commit_info': commit_info,
                'full_content': result.stdout
            }
            
        except subprocess.TimeoutExpired:
            logger.error(f"Git command timed out for file: {path}")
            return {'success': False, 'error': 'Git command timed out'}
        except Exception as e:
            logger.error(f"Error in get_git_commit_info: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def stage_lines(self, path: str, lines: List[int]) -> Dict[str, Any]:
        """
        Stage specific lines of a file.
        
        Args:
            path: File path
            lines: List of line numbers (1-based) to stage
            
        Returns:
            Dictionary with updated git changes
        """
        file_path = Path(path).expanduser().resolve()
        
        try:
            if not lines:
                # Stage entire file
                cmd = ['git', 'add', str(file_path)]
                result = self._run_shell_command(cmd, cwd=file_path.parent)
                if result.returncode != 0:
                    return {'success': False, 'error': result.stderr}
            else:
                # Get the unified diff of unstaged changes
                cmd = ['git', 'diff', '--no-color', '--no-ext-diff', str(file_path)]
                result = self._run_shell_command(cmd, cwd=file_path.parent)
                if result.returncode != 0:
                    return {'success': False, 'error': result.stderr}
                
                if not result.stdout:
                    # No unstaged changes
                    return {
                        'success': True, 
                        'changes': self._get_file_git_changes(file_path)
                    }
                
                # Parse the diff and create a new patch with only the requested lines
                logger.debug(f"Creating patch for lines: {lines}")
                patch = self._create_patch_for_lines(result.stdout, lines)
                
                if not patch:
                    # No changes for the requested lines
                    logger.warning(f"No patch generated for lines {lines} - no matching changes found")
                    return {
                        'success': True,
                        'changes': self._get_file_git_changes(file_path)
                    }
                
                logger.debug(f"Generated patch ({len(patch)} bytes):\n{patch[:500]}")
                
                # Apply the patch to stage the specific lines
                # Try with --recount and --3way for resilience
                cmd = ['git', 'apply', '--cached', '--unidiff-zero', '--recount', '--3way', '-']
                result = self._run_shell_command(cmd, cwd=file_path.parent, input=patch)
                if result.returncode != 0:
                    logger.error(f"Failed to apply patch: {result.stderr}")
                    logger.error(f"Patch was:\n{patch}")
                    return {'success': False, 'error': f"Failed to stage lines: {result.stderr}"}
            
            # Return updated changes
            changes = self._get_file_git_changes(file_path)
            return {'success': True, 'changes': changes}
            
        except Exception as e:
            logger.error(f"Failed to stage lines in {path}: {e}")
            return {'success': False, 'error': str(e)}
    
    @log_proxy_method
    def unstage_lines(self, path: str, lines: List[int]) -> Dict[str, Any]:
        """
        Unstage specific lines of a file.
        
        Args:
            path: File path
            lines: List of line numbers (1-based) to unstage
            
        Returns:
            Dictionary with updated git changes
        """
        file_path = Path(path).expanduser().resolve()
        
        try:
            if not lines:
                # Unstage entire file
                cmd = ['git', 'reset', 'HEAD', str(file_path)]
                result = self._run_shell_command(cmd, cwd=file_path.parent)
                if result.returncode != 0:
                    return {'success': False, 'error': result.stderr}
            else:
                # Get the unified diff of staged changes
                cmd = ['git', 'diff', '--cached', '--no-color', '--no-ext-diff', str(file_path)]
                result = self._run_shell_command(cmd, cwd=file_path.parent)
                if result.returncode != 0:
                    return {'success': False, 'error': result.stderr}
                
                if not result.stdout:
                    # No staged changes
                    return {
                        'success': True,
                        'changes': self._get_file_git_changes(file_path)
                    }
                
                # Parse the diff and create a patch with only the requested lines
                patch = self._create_patch_for_lines(result.stdout, lines)
                
                if not patch:
                    # No changes for the requested lines
                    return {
                        'success': True,
                        'changes': self._get_file_git_changes(file_path)
                    }
                
                # Apply the reverse patch to unstage the specific lines
                cmd = ['git', 'apply', '--cached', '--reverse', '--unidiff-zero', '--recount', '--3way', '-']
                result = self._run_shell_command(cmd, cwd=file_path.parent, input=patch)
                if result.returncode != 0:
                    logger.error(f"Failed to apply reverse patch: {result.stderr}")
                    return {'success': False, 'error': f"Failed to unstage lines: {result.stderr}"}
            
            # Return updated changes
            changes = self._get_file_git_changes(file_path)
            return {'success': True, 'changes': changes}
            
        except Exception as e:
            logger.error(f"Failed to unstage lines in {path}: {e}")
            return {'success': False, 'error': str(e)}

    @log_proxy_method
    def drop_lines(self, path: str, lines: List[int]) -> Dict[str, Any]:
        """
        Revert specific working-tree lines back to HEAD.

        Args:
            path: File path
            lines: List of line numbers (1-based) to revert

        Returns:
            Dictionary with updated git changes
        """
        file_path = Path(path).expanduser().resolve()

        try:
            # Read current file content
            if not file_path.exists():
                return {'success': False, 'error': 'File does not exist'}
            with open(file_path, 'r', encoding='utf-8', errors='replace') as f:
                current_text = f.read()

            # Get HEAD content via existing helper
            head_res = self.get_git_file_version(str(file_path), 'HEAD')
            if not head_res.get('success'):
                return {'success': False, 'error': head_res.get('error', 'Failed to get HEAD version')}
            head_text = head_res.get('content', '')

            # Split into lines preserving newlines
            cur_lines = current_text.splitlines(keepends=True)
            head_lines = head_text.splitlines(keepends=True)

            if not lines:
                # Replace entire file with HEAD
                new_text = head_text
            else:
                # Revert specified working-file lines: replace with HEAD or delete if added
                # Process in descending order to avoid index shifting on deletions
                new_lines = list(cur_lines)
                for ln in sorted(set(lines), reverse=True):
                    idx = ln - 1
                    if idx < 0:
                        continue
                    if idx < len(head_lines):
                        # Replace current line with HEAD's line
                        if idx < len(new_lines):
                            new_lines[idx] = head_lines[idx]
                        else:
                            # Current file shorter than index; append missing HEAD line at end
                            # This case is rare; treat as append
                            # Ensure we fill gaps with newlines if needed
                            while len(new_lines) < idx:
                                new_lines.append('\n')
                            new_lines.append(head_lines[idx])
                    else:
                        # HEAD doesn't have this line -> it was an insertion; remove it if present
                        if idx < len(new_lines):
                            new_lines.pop(idx)
                new_text = ''.join(new_lines)

            # Write updated content back to disk
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(new_text)

            # Report updated changes
            changes = self._get_file_git_changes(file_path)
            return {'success': True, 'changes': changes}

        except Exception as e:
            logger.error(f"Failed to drop lines in {path}: {e}")
            return {'success': False, 'error': str(e)}

    def _create_patch_for_lines(self, diff_text: str, target_lines: List[int]) -> str:
        """
        Create a patch containing only changes for specific lines.
        
        Args:
            diff_text: Git diff output
            target_lines: List of line numbers (1-based) to include in the patch
            
        Returns:
            Patch text containing only the requested lines
        """
        lines = diff_text.split('\n')
        result = []
        file_headers = []
        # We will generate a minimal unidiff-zero patch for each requested new-file line
        # by extracting only the relevant +/- pair for modifications or a single + for additions.
        
        # Convert target_lines to a set for faster lookup
        target_set = set(target_lines)
        
        import re
        i = 0
        while i < len(lines):
            line = lines[i]
            if line.startswith('diff --git'):
                # Capture file headers (needed once at top of patch)
                file_headers = [line]
                i += 1
                while i < len(lines) and not lines[i].startswith('@@'):
                    file_headers.append(lines[i])
                    i += 1
                continue

            if line.startswith('@@'):
                # Parse hunk header: @@ -old_start,old_count +new_start,new_count @@
                m_old = re.search(r'-([0-9]+)(?:,([0-9]+))?', line)
                m_new = re.search(r'\+([0-9]+)(?:,([0-9]+))?', line)
                if not (m_old and m_new):
                    i += 1
                    continue
                old_start = int(m_old.group(1))
                new_start = int(m_new.group(1))
                # Walk hunk lines and build per-line mini hunks
                j = i + 1
                old_ln = old_start
                new_ln = new_start
                # Pre-collect structure for this hunk
                hunk_body = []
                while j < len(lines) and not lines[j].startswith('diff --git') and not lines[j].startswith('@@'):
                    t = lines[j]
                    typ = 'context'
                    if t.startswith(' '):
                        typ = 'context'
                        hunk_body.append((typ, t[1:], old_ln, new_ln))
                        old_ln += 1
                        new_ln += 1
                    elif t.startswith('-'):
                        typ = 'del'
                        hunk_body.append((typ, t[1:], old_ln, new_ln))
                        old_ln += 1
                    elif t.startswith('+'):
                        typ = 'add'
                        hunk_body.append((typ, t[1:], old_ln, new_ln))
                        new_ln += 1
                    else:
                        # Unexpected; treat as context
                        hunk_body.append(('context', t, old_ln, new_ln))
                        old_ln += 1
                        new_ln += 1
                    j += 1

                # First, try to emit grouped hunks for blocks where ALL additions are selected.
                consumed_adds = set()
                # Identify non-context blocks
                bidx = 0
                while bidx < len(hunk_body):
                    if hunk_body[bidx][0] == 'context':
                        bidx += 1
                        continue
                    # Collect block
                    block_dels = []  # List[Tuple[str,int]] (text, old_ln)
                    block_adds = []  # List[Tuple[str,int]] (text, new_ln)
                    start_b = bidx
                    while bidx < len(hunk_body) and hunk_body[bidx][0] != 'context':
                        typ_b, text_b, o_ln_b, n_ln_b = hunk_body[bidx]
                        if typ_b == 'del':
                            block_dels.append((text_b, o_ln_b))
                        elif typ_b == 'add':
                            block_adds.append((text_b, n_ln_b))
                        bidx += 1
                    # Decide grouping
                    if block_adds:
                        add_lines = [n for (_t, n) in block_adds]
                        all_selected = all(n in target_set for n in add_lines)
                        if all_selected:
                            if not result:
                                result.extend(file_headers)
                            if block_dels:
                                old_start_block = block_dels[0][1]
                                old_count_block = len(block_dels)
                                new_start_block = block_adds[0][1]
                                new_count_block = len(block_adds)
                                hdr = f"@@ -{old_start_block},{old_count_block} +{new_start_block},{new_count_block} @@"
                                result.append(hdr)
                                for d_text, _ in block_dels:
                                    result.append(f"-{d_text}")
                                for a_text, _ in block_adds:
                                    result.append(f"+{a_text}")
                            else:
                                # Pure insertion block
                                new_start_block = block_adds[0][1]
                                new_count_block = len(block_adds)
                                hdr = f"@@ -{new_start_block-1},0 +{new_start_block},{new_count_block} @@"
                                result.append(hdr)
                                for a_text, _ in block_adds:
                                    result.append(f"+{a_text}")
                            # Mark additions as consumed so we don't emit per-line hunks
                            for _t, n in block_adds:
                                consumed_adds.add(n)

                # For remaining target additions, build minimal per-line hunks
                for typ, text, o_ln, n_ln in hunk_body:
                    if typ != 'add':
                        continue
                    if n_ln not in target_set:
                        continue
                    if n_ln in consumed_adds:
                        continue
                    # Determine if this is a modification (paired with a '-' just before any context)
                    # Find nearest preceding deletion line without crossing a context boundary
                    k = hunk_body.index((typ, text, o_ln, n_ln))
                    # Backtrack within this hunk
                    pair_old_text = None
                    pair_old_ln = None
                    kk = k - 1
                    while kk >= 0:
                        p_typ, p_text, p_old_ln, p_new_ln = hunk_body[kk]
                        if p_typ == 'context':
                            break
                        if p_typ == 'del':
                            pair_old_text = p_text
                            pair_old_ln = p_old_ln
                            break
                        kk -= 1
                    # Compose a unidiff-zero minimal hunk
                    if pair_old_text is not None:
                        # Modified line: one deletion + one addition
                        hdr = f"@@ -{pair_old_ln},1 +{n_ln},1 @@"
                        body = [f"-{pair_old_text}", f"+{text}"]
                    else:
                        # Added line: no deletions; anchor at t with old count 0
                        # Use old_start as n_ln-1 to position insertion
                        hdr = f"@@ -{n_ln-1},0 +{n_ln},1 @@"
                        body = [f"+{text}"]
                    if not result:
                        result.extend(file_headers)
                    result.append(hdr)
                    result.extend(body)

                i = j
                continue
            i += 1

        return '\n'.join(result) + ('\n' if result else '')
    
    # Removed: previous entire-hunk inclusion; replaced by minimal hunk builder above
    
    def _get_file_git_changes(self, file_path: Path) -> Dict[str, Any]:
        """
        Get comprehensive git changes for a file, including staged and unstaged.
        
        Returns dict with:
        - added, deleted, modified: All changes (staged + unstaged)
        - staged_added, staged_deleted, staged_modified: Only staged changes
        """
        changes = {
            'added': [], 'deleted': [], 'modified': [],
            'staged_added': [], 'staged_deleted': [], 'staged_modified': []
        }
        
        try:
            # Get unstaged changes (working tree vs index)
            cmd_unstaged = ['git', 'diff', '--', str(file_path)]
            result_unstaged = self._run_shell_command(cmd_unstaged, cwd=file_path.parent)
            unstaged = {}
            if result_unstaged.returncode == 0 and result_unstaged.stdout:
                unstaged = self._parse_git_diff(result_unstaged.stdout)
            
            # Get staged changes (index vs HEAD)
            cmd_staged = ['git', 'diff', '--cached', '--', str(file_path)]
            result_staged = self._run_shell_command(cmd_staged, cwd=file_path.parent)
            staged = {}
            if result_staged.returncode == 0 and result_staged.stdout:
                staged = self._parse_git_diff(result_staged.stdout)
                # Store staged changes separately
                changes['staged_added'] = staged.get('added', [])
                changes['staged_deleted'] = staged.get('deleted', [])
                changes['staged_modified'] = staged.get('modified', [])
            
            # Get all changes (working tree vs HEAD)
            cmd_all = ['git', 'diff', 'HEAD', '--', str(file_path)]
            result_all = self._run_shell_command(cmd_all, cwd=file_path.parent)
            if result_all.returncode == 0 and result_all.stdout:
                all_changes = self._parse_git_diff(result_all.stdout)
                changes['added'] = all_changes.get('added', [])
                changes['deleted'] = all_changes.get('deleted', [])
                changes['modified'] = all_changes.get('modified', [])
            else:
                # Fallback: combine staged and unstaged
                for key in ['added', 'deleted', 'modified']:
                    changes[key] = sorted(list(set(
                        staged.get(key, []) + unstaged.get(key, [])
                    )))
        except Exception as e:
            logger.warning(f"Could not get git changes for {file_path}: {e}")
        
        return changes
    
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

        run_added = []
        run_deleted = []

        def flush_run() -> None:
            nonlocal run_added, run_deleted
            if not run_added and not run_deleted:
                return
            if run_added and run_deleted:
                modified_lines.extend(run_added)
            else:
                added_lines.extend(run_added)
                deleted_lines.extend(run_deleted)
            run_added = []
            run_deleted = []

        for line in diff_output.split('\n'):
            # Parse hunk header: @@ -old_start,old_count +new_start,new_count @@
            hunk_match = re.match(r'^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@', line)
            if hunk_match:
                flush_run()
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
                run_added.append(current_new_line)
                current_new_line += 1
            elif line.startswith('-') and not line.startswith('---'):
                # Deleted line (mark it at the next line position)
                run_deleted.append(current_old_line)
                current_old_line += 1
            elif line.startswith(' '):
                # Unchanged line
                flush_run()
                current_old_line += 1
                current_new_line += 1
        flush_run()

        return {
            'added': sorted(list(set(added_lines))),
            'deleted': sorted(deleted_lines),
            'modified': sorted(list(set(modified_lines))),
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
    # Parse command-line arguments
    parser = argparse.ArgumentParser(description='Veda Proxy - JSON RPC service')
    parser.add_argument('--test-lsp-python', help='Override Python LSP server command for testing')
    args = parser.parse_args()

    # Build test overrides dictionary from parsed arguments
    test_lsp_overrides = {}
    if args.test_lsp_python:
        test_lsp_overrides['python'] = args.test_lsp_python

    # Emit version handshake as the very first line on stdout
    try:
        print(json.dumps({"version": PROXY_VERSION}), flush=True)
    except Exception:
        # If this fails, continue; client will time out and handle copying
        pass

    proxy = VedaProxy(test_lsp_overrides=test_lsp_overrides if test_lsp_overrides else None)
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
