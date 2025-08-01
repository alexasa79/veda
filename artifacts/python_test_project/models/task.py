"""
Task model representing a single task in the system.
"""

from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum, auto
from typing import Optional, List, Set
from uuid import uuid4


class Priority(Enum):
    """Task priority levels."""
    LOW = auto()
    MEDIUM = auto()
    HIGH = auto()
    CRITICAL = auto()
    
    def __str__(self) -> str:
        return self.name.capitalize()


class Status(Enum):
    """Task status states."""
    TODO = "todo"
    IN_PROGRESS = "in_progress"
    DONE = "done"
    CANCELLED = "cancelled"
    
    @classmethod
    def from_string(cls, value: str) -> 'Status':
        """Create Status from string value."""
        for status in cls:
            if status.value == value.lower():
                return status
        raise ValueError(f"Invalid status: {value}")


@dataclass
class Task:
    """
    Represents a task with various attributes.
    
    Attributes:
        title: The task title
        description: Detailed description
        priority: Task priority level
        status: Current task status
        created_at: Creation timestamp
        due_date: Optional due date
        completed_at: Completion timestamp
        tags: Set of tags for categorization
        assignee: Person assigned to the task
    """
    
    title: str
    description: str = ""
    priority: Priority = Priority.MEDIUM
    status: Status = Status.TODO
    created_at: datetime = field(default_factory=datetime.now)
    due_date: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    tags: Set[str] = field(default_factory=set)
    assignee: Optional[str] = None
    id: str = field(default_factory=lambda: str(uuid4()))
    
    def __post_init__(self):
        """Validate task data after initialization."""
        if not self.title:
            raise ValueError("Task title cannot be empty")
        
        if self.due_date and self.due_date < self.created_at:
            raise ValueError("Due date cannot be before creation date")
    
    def complete(self) -> None:
        """Mark the task as completed."""
        if self.status == Status.DONE:
            raise ValueError("Task is already completed")
        
        self.status = Status.DONE
        self.completed_at = datetime.now()
    
    def start(self) -> None:
        """Mark the task as in progress."""
        if self.status == Status.DONE:
            raise ValueError("Cannot start a completed task")
        
        self.status = Status.IN_PROGRESS
    
    def cancel(self) -> None:
        """Cancel the task."""
        if self.status == Status.DONE:
            raise ValueError("Cannot cancel a completed task")
        
        self.status = Status.CANCELLED
    
    def add_tag(self, tag: str) -> None:
        """Add a tag to the task."""
        self.tags.add(tag.lower())
    
    def remove_tag(self, tag: str) -> None:
        """Remove a tag from the task."""
        self.tags.discard(tag.lower())
    
    def is_overdue(self) -> bool:
        """Check if the task is overdue."""
        if not self.due_date or self.status == Status.DONE:
            return False
        return datetime.now() > self.due_date
    
    def days_until_due(self) -> Optional[int]:
        """Calculate days until due date."""
        if not self.due_date:
            return None
        
        delta = self.due_date - datetime.now()
        return delta.days
    
    @property
    def is_completed(self) -> bool:
        """Check if task is completed."""
        return self.status == Status.DONE
    
    def __repr__(self) -> str:
        return f"Task(id={self.id[:8]}, title='{self.title}', status={self.status.value})"