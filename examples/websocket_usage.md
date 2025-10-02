# User-Specific WebSocket Log Streaming with Message-Based Authentication

## Overview

The WebSocket log streaming has been updated to support user-specific log streams with **message-based authentication** suitable for browser clients. Each authenticated user gets their own isolated log stream, ensuring that users only see logs relevant to their sessions.

## Authentication Flow

Since browsers cannot send custom headers during WebSocket handshake, we use message-based authentication:

1. **Connect**: Client establishes WebSocket connection to `/logsws`
2. **Authenticate**: Client sends authentication message as first message
3. **Respond**: Server validates token and responds with success/failure
4. **Stream**: If authenticated, server starts streaming user-specific logs

## Message Protocol

### Authentication Message (Client → Server)
```json
{
  "type": "auth",
  "data": {
    "token": "your-bearer-token-here"
  }
}
```

### Authentication Response (Server → Client)
```json
// Success
{
  "type": "auth_response",
  "data": {
    "success": true,
    "user_id": "john.doe@company.com"
  }
}

// Failure
{
  "type": "auth_response",
  "data": {
    "success": false,
    "error": "Authentication failed: invalid token"
  }
}
```

### Log Messages (Server → Client)
```json
{
  "type": "logs",
  "data": {
    "logs": "Your log content here..."
  }
}
```

### Error Messages (Server → Client)
```json
{
  "type": "error",
  "data": {
    "message": "Error description"
  }
}
```

## API Endpoints

### WebSocket Connection
```
GET /logsws
```

The WebSocket will:
1. Wait for authentication message from client
2. Validate the token and extract user ID
3. Send authentication response
4. If successful, send initial logs for that user
5. Stream real-time updates for that user only

### HTTP Endpoints (with optional user parameter)

#### Get Logs
```
GET /logs?user=<userID>    # Get logs for specific user
GET /logs                  # Get global logs (backward compatibility)
```

#### Append Logs
```
PUT /logs/append?user=<userID>    # Append to user-specific logs
PUT /logs/append                  # Append to global logs
```

#### Set Logs
```
PUT /logs?user=<userID>    # Set logs for specific user
PUT /logs                  # Set global logs
```

#### Delete Logs
```
DELETE /logs?user=<userID>    # Clear logs for specific user
DELETE /logs                  # Clear global logs
```

## Redis Key Structure

- Global logs: `logs`
- User-specific logs: `logs:<userID>`
- Global pub/sub channel: `redis-log-stream-pubsub-channel`
- User-specific pub/sub channels: `redis-log-stream-pubsub-channel:<userID>`

## Example Usage

### JavaScript WebSocket Client (Browser-Compatible)
```javascript
// Establish WebSocket connection
const ws = new WebSocket('ws://localhost:8881/logsws');
let authenticated = false;
let userID = null;

ws.onopen = function() {
  console.log('WebSocket connected, sending authentication...');
  
  // Send authentication message
  const authMessage = {
    type: 'auth',
    data: {
      token: 'your-bearer-token-here' // Get this from your auth system
    }
  };
  
  ws.send(JSON.stringify(authMessage));
};

ws.onmessage = function(event) {
  const message = JSON.parse(event.data);
  
  switch(message.type) {
    case 'auth_response':
      if (message.data.success) {
        authenticated = true;
        userID = message.data.user_id;
        console.log('Authentication successful for user:', userID);
      } else {
        console.error('Authentication failed:', message.data.error);
        ws.close();
      }
      break;
      
    case 'logs':
      console.log('Received logs:', message.data.logs);
      // Update your UI with the logs
      document.getElementById('logs').textContent = message.data.logs;
      break;
      
    case 'error':
      console.error('Server error:', message.data.message);
      break;
      
    default:
      console.warn('Unknown message type:', message.type);
  }
};

ws.onerror = function(error) {
  console.error('WebSocket error:', error);
};

ws.onclose = function(event) {
  console.log('WebSocket closed:', event.code, event.reason);
  authenticated = false;
  userID = null;
};

// Function to check if connected and authenticated
function isReady() {
  return ws.readyState === WebSocket.OPEN && authenticated;
}
```

### React Hook Example
```javascript
import { useState, useEffect, useRef } from 'react';

function useLogStream(token) {
  const [logs, setLogs] = useState('');
  const [connected, setConnected] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [error, setError] = useState(null);
  const [userID, setUserID] = useState(null);
  const ws = useRef(null);

  useEffect(() => {
    if (!token) return;

    // Create WebSocket connection
    ws.current = new WebSocket('ws://localhost:8881/logsws');

    ws.current.onopen = () => {
      setConnected(true);
      setError(null);
      
      // Send authentication
      const authMessage = {
        type: 'auth',
        data: { token }
      };
      ws.current.send(JSON.stringify(authMessage));
    };

    ws.current.onmessage = (event) => {
      const message = JSON.parse(event.data);
      
      switch(message.type) {
        case 'auth_response':
          if (message.data.success) {
            setAuthenticated(true);
            setUserID(message.data.user_id);
            setError(null);
          } else {
            setError(message.data.error);
            setAuthenticated(false);
          }
          break;
          
        case 'logs':
          setLogs(message.data.logs);
          break;
          
        case 'error':
          setError(message.data.message);
          break;
      }
    };

    ws.current.onerror = (error) => {
      setError('WebSocket connection error');
      setConnected(false);
      setAuthenticated(false);
    };

    ws.current.onclose = () => {
      setConnected(false);
      setAuthenticated(false);
      setUserID(null);
    };

    // Cleanup on unmount
    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [token]);

  return { logs, connected, authenticated, error, userID };
}

// Usage in component
function LogViewer({ authToken }) {
  const { logs, connected, authenticated, error, userID } = useLogStream(authToken);

  if (error) {
    return <div className="error">Error: {error}</div>;
  }

  if (!connected) {
    return <div>Connecting...</div>;
  }

  if (!authenticated) {
    return <div>Authenticating...</div>;
  }

  return (
    <div>
      <h3>Logs for {userID}</h3>
      <pre className="logs">{logs}</pre>
    </div>
  );
}
```

### HTTP API Usage
```bash
# Get logs for a specific user
curl -H "Authorization: Bearer <token>" \
     "http://localhost:8881/logs?user=john.doe@company.com"

# Append logs for a specific user
curl -X PUT \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '"New log entry"' \
     "http://localhost:8881/logs/append?user=john.doe@company.com"
```

## Security Features

1. **30-second authentication timeout**: Clients must authenticate within 30 seconds of connection
2. **Token validation**: Server validates bearer tokens using existing auth infrastructure
3. **User isolation**: Each user only receives their own logs
4. **Error handling**: Graceful error responses for authentication failures

## Benefits

1. **Browser Compatible**: No custom headers required - works with all browser WebSocket implementations
2. **User Isolation**: Each user only sees their own logs
3. **Scalability**: Multiple users can use the system simultaneously without interference
4. **Security**: Users cannot access other users' log data
5. **Backward Compatibility**: Existing HTTP APIs continue to work with global logs
6. **Real-time Updates**: WebSocket connections provide immediate log updates for each user
7. **Structured Protocol**: Clear message types for easy client implementation