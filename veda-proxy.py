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
from typing import Dict, List, Any, Optional
import logging
import subprocess
import threading
import queue

# Configure logging to file so it doesn't interfere with JSON RPC communication
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - %(message)s',
    filename='veda-proxy.log',
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
            }
        }
        
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
        
        request = {
            "jsonrpc": "2.0",
            "id": server['request_id'],
            "method": "initialize",
            "params": {
                "processId": os.getpid(),
                "rootUri": f"file://{Path.cwd().absolute()}",
                "capabilities": {
                    "textDocument": {
                        "definition": {
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
    
    def _open_file(self, language: str, file_path: str):
        """Notify LSP server that a file has been opened."""
        if language not in self.servers:
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
            '.hxx': 'cpp'
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
            # Diagnostics - we'll handle these later
            pass
    
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


class VedaProxy:
    """Main proxy class handling JSON RPC requests for various operations."""

    def __init__(self):
        self.lsp = LSPManager()
        self.methods = {
            'list_directory': self.list_directory,
            'list_directory_recursive': self.list_directory_recursive,
            'get_file': self.get_file,
            'start_lsp_server': self.start_lsp_server,
            'stop_lsp_server': self.stop_lsp_server,
            'go_to_definition': self.go_to_definition,
            'get_lsp_messages': self.get_lsp_messages,
            'get_lsp_notifications': self.get_lsp_notifications,
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
        
        def _walk_directory(current_path: Path):
            try:
                for entry in current_path.iterdir():
                    try:
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
            content = file_path.read_text(encoding='utf-8')
            
            # Notify LSP server about the opened file
            language = self.lsp.get_language_for_file(str(file_path))
            if language:
                # Start LSP server if needed
                if language not in self.lsp.servers:
                    logger.info(f"Starting LSP server for {language} when opening {path}")
                    self.lsp.start_server(language)
                
                # Notify LSP about the opened file
                if language in self.lsp.servers:
                    self.lsp._open_file(language, str(file_path))
                    logger.info(f"Notified LSP server about opened file: {path}")
            
            return {'content': content}
        except UnicodeDecodeError:
            # Try reading as binary and return error
            raise ValueError(f"File is not a text file: {path}")
        except PermissionError:
            raise PermissionError(f"Permission denied: {path}")

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
                print(json.dumps(response), flush=True)

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
