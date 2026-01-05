#!/usr/bin/env python3
"""
CLI helper for postgres-test-replay API
Makes it easier to interact with the IPC service
"""

import sys
import json
import argparse
import requests
from datetime import datetime

API_BASE = "http://localhost:8080/api"

def print_json(data):
    """Pretty print JSON data"""
    print(json.dumps(data, indent=2))

def create_session(name, description, database):
    """Create a new session"""
    data = {
        "name": name,
        "description": description,
        "database": database
    }
    response = requests.post(f"{API_BASE}/sessions", json=data)
    response.raise_for_status()
    print("✓ Session created:")
    print_json(response.json())

def list_sessions():
    """List all sessions"""
    response = requests.get(f"{API_BASE}/sessions")
    response.raise_for_status()
    sessions = response.json()
    print(f"Found {len(sessions)} session(s):")
    print_json(sessions)

def switch_session(session_id):
    """Switch to a different session"""
    data = {"session_id": session_id}
    response = requests.post(f"{API_BASE}/sessions/switch", json=data)
    response.raise_for_status()
    print(f"✓ Switched to session: {session_id}")

def create_checkpoint(name, description, lsn, entry_index, session_id):
    """Create a checkpoint"""
    data = {
        "name": name,
        "description": description,
        "lsn": lsn,
        "entry_index": entry_index,
        "session_id": session_id
    }
    response = requests.post(f"{API_BASE}/checkpoints", json=data)
    response.raise_for_status()
    print("✓ Checkpoint created:")
    print_json(response.json())

def list_checkpoints(session_id=None):
    """List checkpoints"""
    params = {}
    if session_id:
        params['session_id'] = session_id
    response = requests.get(f"{API_BASE}/checkpoints", params=params)
    response.raise_for_status()
    checkpoints = response.json()
    print(f"Found {len(checkpoints)} checkpoint(s):")
    print_json(checkpoints)

def navigate_to_checkpoint(checkpoint_id):
    """Navigate to a checkpoint"""
    data = {"checkpoint_id": checkpoint_id}
    response = requests.post(f"{API_BASE}/navigate", json=data)
    response.raise_for_status()
    result = response.json()
    print(f"✓ Retrieved {result['count']} entries")
    if result['count'] < 10:
        print_json(result['entries'])

def replay_session(session_id, checkpoint_id):
    """Replay a session up to a checkpoint"""
    data = {
        "session_id": session_id,
        "checkpoint_id": checkpoint_id
    }
    response = requests.post(f"{API_BASE}/replay", json=data)
    response.raise_for_status()
    result = response.json()
    print("✓ Replay completed:")
    print_json(result)

def health_check():
    """Check API health"""
    response = requests.get(f"http://localhost:8080/health")
    response.raise_for_status()
    print("✓ API is healthy")
    print_json(response.json())

def main():
    parser = argparse.ArgumentParser(description='PostgreSQL Test Replay CLI')
    subparsers = parser.add_subparsers(dest='command', help='Command to execute')

    # Session commands
    session_create = subparsers.add_parser('session-create', help='Create a session')
    session_create.add_argument('name', help='Session name')
    session_create.add_argument('--description', default='', help='Session description')
    session_create.add_argument('--database', default='testdb', help='Database name')

    subparsers.add_parser('session-list', help='List sessions')

    session_switch = subparsers.add_parser('session-switch', help='Switch session')
    session_switch.add_argument('session_id', help='Session ID')

    # Checkpoint commands
    checkpoint_create = subparsers.add_parser('checkpoint-create', help='Create a checkpoint')
    checkpoint_create.add_argument('name', help='Checkpoint name')
    checkpoint_create.add_argument('--description', default='', help='Checkpoint description')
    checkpoint_create.add_argument('--lsn', default='0/0', help='LSN position')
    checkpoint_create.add_argument('--entry-index', type=int, default=0, help='Entry index')
    checkpoint_create.add_argument('--session-id', required=True, help='Session ID')

    checkpoint_list = subparsers.add_parser('checkpoint-list', help='List checkpoints')
    checkpoint_list.add_argument('--session-id', help='Filter by session ID')

    # Navigate commands
    navigate = subparsers.add_parser('navigate', help='Navigate to checkpoint')
    navigate.add_argument('checkpoint_id', help='Checkpoint ID')

    # Replay commands
    replay = subparsers.add_parser('replay', help='Replay session')
    replay.add_argument('session_id', help='Session ID')
    replay.add_argument('checkpoint_id', help='Checkpoint ID')

    # Health check
    subparsers.add_parser('health', help='Check API health')

    args = parser.parse_args()

    try:
        if args.command == 'session-create':
            create_session(args.name, args.description, args.database)
        elif args.command == 'session-list':
            list_sessions()
        elif args.command == 'session-switch':
            switch_session(args.session_id)
        elif args.command == 'checkpoint-create':
            create_checkpoint(args.name, args.description, args.lsn, 
                            args.entry_index, args.session_id)
        elif args.command == 'checkpoint-list':
            list_checkpoints(args.session_id)
        elif args.command == 'navigate':
            navigate_to_checkpoint(args.checkpoint_id)
        elif args.command == 'replay':
            replay_session(args.session_id, args.checkpoint_id)
        elif args.command == 'health':
            health_check()
        else:
            parser.print_help()
    except requests.exceptions.RequestException as e:
        print(f"✗ Error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"✗ Unexpected error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
