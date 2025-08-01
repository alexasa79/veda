"""
Input validation and sanitization utilities.
"""

import re
from datetime import datetime
from typing import Optional


def validate_email(email: str) -> bool:
    """
    Validate email address format.
    
    Args:
        email: Email address to validate
        
    Returns:
        True if valid email format, False otherwise
    """
    if not email or not isinstance(email, str):
        return False
    
    # Basic email pattern
    pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    return bool(re.match(pattern, email.strip()))


def validate_date_string(date_str: str, format: str = "%Y-%m-%d") -> bool:
    """
    Validate date string format.
    
    Args:
        date_str: Date string to validate
        format: Expected date format (default: YYYY-MM-DD)
        
    Returns:
        True if valid date format, False otherwise
    """
    if not date_str or not isinstance(date_str, str):
        return False
    
    try:
        datetime.strptime(date_str.strip(), format)
        return True
    except ValueError:
        return False


def validate_task_title(title: str, min_length: int = 3, max_length: int = 200) -> bool:
    """
    Validate task title.
    
    Args:
        title: Task title to validate
        min_length: Minimum allowed length
        max_length: Maximum allowed length
        
    Returns:
        True if valid title, False otherwise
    """
    if not title or not isinstance(title, str):
        return False
    
    stripped_title = title.strip()
    if not stripped_title:
        return False
    
    return min_length <= len(stripped_title) <= max_length


def sanitize_string(text: str, max_length: Optional[int] = None) -> str:
    """
    Sanitize user input string.
    
    Args:
        text: Text to sanitize
        max_length: Maximum allowed length (optional)
        
    Returns:
        Sanitized string
    """
    if not isinstance(text, str):
        return ""
    
    # Remove leading/trailing whitespace
    sanitized = text.strip()
    
    # Remove control characters
    sanitized = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', sanitized)
    
    # Normalize whitespace
    sanitized = re.sub(r'\s+', ' ', sanitized)
    
    # Truncate if needed
    if max_length and len(sanitized) > max_length:
        sanitized = sanitized[:max_length]
    
    return sanitized


def validate_priority(priority: str) -> bool:
    """
    Validate priority value.
    
    Args:
        priority: Priority value to validate
        
    Returns:
        True if valid priority, False otherwise
    """
    valid_priorities = ['low', 'medium', 'high', 'critical']
    return priority.lower() in valid_priorities


def validate_status(status: str) -> bool:
    """
    Validate status value.
    
    Args:
        status: Status value to validate
        
    Returns:
        True if valid status, False otherwise
    """
    valid_statuses = ['todo', 'in_progress', 'done', 'cancelled']
    return status.lower() in valid_statuses


def validate_uuid(uuid_str: str) -> bool:
    """
    Validate UUID format.
    
    Args:
        uuid_str: UUID string to validate
        
    Returns:
        True if valid UUID format, False otherwise
    """
    if not uuid_str or not isinstance(uuid_str, str):
        return False
    
    # UUID v4 pattern
    pattern = r'^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$'
    return bool(re.match(pattern, uuid_str.lower()))


def validate_tag(tag: str, max_length: int = 50) -> bool:
    """
    Validate tag format.
    
    Args:
        tag: Tag to validate
        max_length: Maximum allowed length
        
    Returns:
        True if valid tag, False otherwise
    """
    if not tag or not isinstance(tag, str):
        return False
    
    stripped_tag = tag.strip()
    if not stripped_tag:
        return False
    
    # Tags should be alphanumeric with underscores/hyphens
    if not re.match(r'^[a-zA-Z0-9_-]+$', stripped_tag):
        return False
    
    return len(stripped_tag) <= max_length


def sanitize_html(text: str) -> str:
    """
    Basic HTML sanitization to prevent XSS.
    
    Args:
        text: Text to sanitize
        
    Returns:
        Sanitized text with HTML entities escaped
    """
    if not isinstance(text, str):
        return ""
    
    # Basic HTML entity escaping
    replacements = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#x27;',
        '/': '&#x2F;'
    }
    
    sanitized = text
    for char, entity in replacements.items():
        sanitized = sanitized.replace(char, entity)
    
    return sanitized