# Web UI Screenshots and Features

## Main Dashboard

The web UI provides a comprehensive dashboard for managing the PostgreSQL Test Replay system.

### Layout Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│  🔄 PostgreSQL Test Replay                                           │
│  Time-travel debugging and checkpoint replay system                  │
└─────────────────────────────────────────────────────────────────────┘

┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ 📥 Input Database │ │ 📤 Output Database│ │ 📊 Statistics    │
│ ● Connected      │ │ ● Connected       │ │                  │
│ postgres://...   │ │ postgres://...    │ │  5    │  12      │
│                  │ │                   │ │ Logs  │ CPs      │
└──────────────────┘ └──────────────────┘ └──────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│ 🎯 Fixed Checkpoints                                                 │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ 🏁 Database Creation [FIXED]                                    │ │
│ │    Initial state when database was created                      │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ 🔧 Initial Migration [FIXED]                                    │ │
│ │    State after initial schema migration                         │ │
│ └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│ 📌 Checkpoints                         [🔄 Refresh] [➕ Create]      │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ Before Migration                                                │ │
│ │ State before running schema changes                             │ │
│ │ Created: 2024-01-05 10:30:00                                    │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ After Insert                                                    │ │
│ │ Added test data                                                 │ │
│ │ Created: 2024-01-05 10:35:00                                    │ │
│ └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│ 📝 WAL Log Entries      [🔄] [⬆️ Top] [⬇️ Bottom]                    │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ [INSERT] 2024-01-05 10:35:12                                    │ │
│ │ Table: public.test_table | LSN: 0/1234567                       │ │
│ │ {"id": 1, "name": "test", "value": 100}                         │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ [UPDATE] 2024-01-05 10:36:05                                    │ │
│ │ Table: public.test_table | LSN: 0/1234568                       │ │
│ │ {"id": 1, "name": "test", "value": 200}                         │ │
│ └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

## Color Scheme

- **Header**: Purple gradient (modern, professional)
- **Cards**: White with subtle shadows
- **DSN Display**: Light gray background with blue left border
- **Fixed Checkpoints**: Yellow badge indicating permanent status
- **Operations**:
  - INSERT: Green background
  - UPDATE: Blue background
  - DELETE: Red background
  - DDL: Orange background

## Interactive Features

### 1. Database Status Cards
- Real-time connection indicators (green pulsing dot)
- Full DSN strings displayed in monospace font
- Visual separation for Input vs Output databases

### 2. Fixed Checkpoints
- Always visible at the top
- Yellow "FIXED" badge for easy identification
- Clickable to show informational dialog
- Cannot be deleted or modified

### 3. Checkpoint Management
- List of all user-created checkpoints
- Click any checkpoint to navigate to that point
- "Create Checkpoint" button opens prompt dialog
- Shows creation timestamp
- Displays name and description

### 4. WAL Log Viewer
- Scrollable container with recent entries
- Color-coded operation types
- Shows timestamp, table, LSN
- JSON formatted data display
- Auto-refresh every 5 seconds
- Scroll controls for navigation

### 5. Statistics Dashboard
- Live count of WAL log entries
- Live count of checkpoints (including fixed)
- Large, bold numbers for visibility

## User Interactions

### Creating a Checkpoint
1. Click "➕ Create Checkpoint" button
2. Enter checkpoint name in dialog
3. Enter optional description
4. Checkpoint is created and appears in list

### Navigating to Checkpoint
1. Click any checkpoint in the list
2. Confirmation dialog appears
3. System loads WAL entries up to that checkpoint
4. Log viewer updates with relevant entries

### Viewing Fixed Checkpoints
1. Click on "Database Creation" or "Initial Migration"
2. Informational dialog explains the checkpoint
3. Shows what state would be restored

### Scrolling Logs
1. Click "⬆️ Scroll to Top" to jump to beginning
2. Click "⬇️ Scroll to Bottom" to see latest entries
3. Manual scrolling also available

### Monitoring Changes
- Page automatically refreshes data every 5 seconds
- New WAL entries appear automatically
- New checkpoints show up in list
- No manual refresh needed for monitoring

## Technical Details

### Browser Compatibility
- Works in all modern browsers (Chrome, Firefox, Safari, Edge)
- No external dependencies
- Pure HTML, CSS, and JavaScript
- Responsive design for different screen sizes

### Performance
- Efficient API calls with pagination
- Limited number of logs shown (50 by default)
- Auto-refresh with reasonable interval (5 seconds)
- Minimal resource usage

### Accessibility
- Clear visual hierarchy
- Color-coded with meaningful labels
- Large click targets
- Readable fonts and spacing

## URL Structure

- **Main UI**: `http://localhost:8080/`
- **Health Check**: `http://localhost:8080/health`
- **Configuration**: `http://localhost:8080/api/config`
- **WAL Logs**: `http://localhost:8080/api/wal-logs`
- **Checkpoints**: `http://localhost:8080/api/checkpoints`
- **Sessions**: `http://localhost:8080/api/sessions`

## Security Considerations

⚠️ **Important**: The UI displays full DSN strings including credentials. 

**Recommendations**:
- Do not expose the UI publicly
- Use firewall rules to restrict access
- Consider authentication in production
- Use environment variables for sensitive data
- DSN display could be masked in production builds

## Future Enhancements

Potential improvements for the UI:
- Authentication/login system
- Real-time WebSocket updates instead of polling
- Ability to edit/delete checkpoints from UI
- Session switching interface
- Replay progress indicators
- Diff viewer between checkpoints
- Export/import checkpoint data
- Search and filter WAL logs
- Dark mode theme
