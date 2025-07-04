#!/usr/bin/env python3
"""
Vidya Proxy - A JSON RPC service for various operations including file system access.
Communicates via stdin/stdout and can operate locally or remotely via SSH.
"""

import json
import sys
import os
import stat
import pwd
import grp
from pathlib import Path
from typing import Dict, List, Any, Optional
import logging

# Configure logging to stderr so it doesn't interfere with JSON RPC communication
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - %(message)s',
    stream=sys.stderr
)
logger = logging.getLogger(__name__)


class VidyaProxy:
    """Main proxy class handling JSON RPC requests for various operations."""

    def __init__(self):
        self.methods = {
            'list_directory': self.list_directory,
            'get_file_info': self.get_file_info,
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
                file_info = self._get_file_info(entry)
                entries.append(file_info)
            except (OSError, PermissionError) as e:
                logger.warning(f"Could not get info for {entry}: {e}")
                # Add basic info even if we can't get all details
                entries.append({
                    'name': entry.name,
                    'path': str(entry),
                    'type': 'unknown',
                    'error': str(e)
                })

        # Sort entries: directories first, then files, both alphabetically
        entries.sort(key=lambda x: (x.get('type') != 'directory', x['name'].lower()))

        return entries

    def get_file_info(self, path: str) -> Dict[str, Any]:
        """
        Get detailed information about a specific file or directory.

        Args:
            path: File or directory path

        Returns:
            Dictionary with file information
        """
        file_path = Path(path).expanduser().resolve()
        return self._get_file_info(file_path)

    def _get_file_info(self, path: Path) -> Dict[str, Any]:
        """
        Internal method to get detailed file information.

        Args:
            path: Path object

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
            'path': str(path),
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


class JSONRPCHandler:
    """Handles JSON RPC communication over stdin/stdout."""

    def __init__(self, proxy: VidyaProxy):
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
        logger.info("Vidya Proxy started")

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
    proxy = VidyaProxy()
    handler = JSONRPCHandler(proxy)
    handler.run()


if __name__ == '__main__':
    main()
