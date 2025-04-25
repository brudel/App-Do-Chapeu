# MultiTag Sync App - Plan

## Goal
Create a Flutter application and a Go server to synchronize the display of a pre-loaded image on multiple (max 10) devices after all users signal readiness. The system must be resilient to minor network instability.

## Core Concepts
*   **Client ID:** Each app instance generates and persists a unique UUID for identification across connections.
*   **State Reconciliation:** Clients and server sync state via WebSocket messages.
*   **Image Preloading/Caching:** The single active image is downloaded/cached by clients as soon as possible.
*   **Target Time Synchronization:** The server dictates a future UTC timestamp for synchronized actions (vibration, loading, display) to mitigate latency variations.

## Phase 1: Server Implementation (Go with Gin & Gorilla WebSocket)

1.  **Project Setup:**
    *   Directory: `server/`
    *   `go mod init <your_module_name>`
    *   Dependencies: `github.com/gin-gonic/gin`, `github.com/gorilla/websocket`, `github.com/google/uuid`
2.  **Data Structures (Protected by Mutexes):**
    *   `ClientState`: `{ ClientID string; IsReady bool; LastSeen time.Time; WSConn *websocket.Conn }`
    *   `ServerState`: `{ Clients map[string]*ClientState; ExpectedUsers int; // Potentially dynamic or configured CurrentImageExists bool; // Flag indicating if an image has been uploaded OverallState string; // e.g., "WaitingForUsers", "WaitingForReady", "Triggered", "Displaying" TargetShowTimeUTC time.Time }`
3.  **Gin Setup (main.go):**
    *   Router: `r := gin.Default()`
    *   Routes:
        *   `POST /upload`: Image upload handler.
        *   `GET /image`: Fixed image serving handler.
        *   `GET /ws`: WebSocket upgrade handler.
    *   Run: `r.Run(":8080")` (or configured port)
4.  **Handlers:**
    *   **WebSocket Handler (`/ws`):**
        *   Upgrade connection.
        *   **Reader Loop:**
            *   Expect initial `{"type": "register", "clientId": "uuid-string"}`. Register/update client, store `WSConn`, send current `full_state`.
            *   Handle `{"type": "ready", "isReady": bool}`. Update client state, broadcast `partial_state`. Check if all ready:
                *   If yes: Set `OverallState` to `"Triggered"`, calculate `TargetShowTimeUTC` (e.g., Now + 5s), broadcast `{"type": "start", "targetTimestampUTC": TargetTime}`.
        *   **Disconnection:** Clear `WSConn`, update `LastSeen`. Implement cleanup for inactive clients.
    *   **Image Upload (`/upload`):**
        *   Accept `multipart/form-data`.
        *   Save file to a fixed path (e.g., `uploads/current_image.png`, overwriting).
        *   Set `ServerState.CurrentImageExists = true`.
        *   Broadcast `{"type": "image_updated"}` to all connected clients (signals them to re-fetch/validate cache for `/image`).
    *   **Image Serve (`/image`):**
        *   Serve the fixed file (e.g., `uploads/current_image.png`). Handle case where file doesn't exist yet (404).
5.  **Concurrency:** Use `sync.Mutex` for safe access to `ServerState`.

## Phase 2: Flutter App Implementation (mobile/)

1.  **Dependencies:**
    *   `web_socket_channel`, `http`, `image_picker` (admin)
    *   `uuid`, `shared_preferences` (client ID)
    *   `path_provider`, `flutter_cache_manager` (image caching)
    *   `vibrator`, `intl` (timestamps), `connectivity_plus`
2.  **Initialization:** Generate/load persistent `clientId` (UUID).
3.  **State Management (Client):** Store `clientId`, connection status, server state (`readyCount`, `totalCount`, `overallState`, `targetShowTimeUTC`), local readiness, cached image status.
4.  **WebSocket Service:**
    *   Connect to `/ws`, send `register`.
    *   Implement robust reconnection logic (backoff, use `connectivity_plus`).
    *   **Message Handling:**
        *   `full_state`: Update local state. Trigger image cache validation/fetch for `/image` if needed.
        *   `partial_state`: Update readiness counts.
        *   `image_updated`: Invalidate cache and trigger background re-fetch for `/image`.
        *   `start`: Store `targetShowTimeUTC`. Schedule local actions (vibrate, loading, show image) via `Timer` for the target time. Ensure `/image` is cached.
5.  **Image Caching:**
    *   Use `flutter_cache_manager` for the fixed `/image` URL. Re-fetch/validate when `image_updated` is received or state requires it.
6.  **UI:** Reflect state, ready toggle, admin upload button. Display cached image at scheduled time.
7.  **Synchronized Actions:**
    *   On `start` message: Calculate `Duration` until target UTC time.
    *   If positive duration: `Timer(duration, () => { Vibrate -> Show Loading -> Future.delayed(3s, () => Show Cached Image) })`.
    *   Handle negative duration (missed target) appropriately.

## Phase 3: Deployment

1.  **Server:** Build Go executable, optionally containerize (Podman), deploy to GCP VM, open firewall port, run as service.
2.  **Flutter App:** Update server URL, build release (`apk`/`ipa`), distribute.

## Flow Diagram (Mermaid)

```mermaid
sequenceDiagram
    participant ClientApp as Flutter App (Client ID)
    participant GoServer as Go Server (Gin + WS)

    ClientApp->>ClientApp: Generate/Load Client ID
    ClientApp->>GoServer: Connect /ws
    GoServer-->>ClientApp: WebSocket Upgrade OK
    ClientApp->>GoServer: Send {"type": "register", "clientId": "id-123"}
    GoServer-->>ClientApp: Send {"type": "full_state", "state": {..., "state": "WaitingForReady"}}

    ClientApp->>ClientApp: Trigger Image Cache Check/Fetch for GET /image

    opt Admin Uploads New Image
        AdminApp->>GoServer: POST /upload (image file) -> saves as uploads/current_image.png
        GoServer->>All Connected Apps: Broadcast {"type": "image_updated"}
        ClientApp->>ClientApp: Invalidate Cache & Trigger Re-fetch for GET /image
    end

    ClientApp->>GoServer: Send {"type": "ready", "isReady": true}
    GoServer->>GoServer: Update Client "id-123" status
    GoServer->>All Connected Apps: Broadcast {"type": "partial_state", "clientId": "id-123", "isReady": true, "readyCount": X}

    Note over GoServer: Last user ready, calculate Target Time (Now + 5s)
    GoServer->>All Connected Apps: Broadcast {"type": "start", "targetTimestampUTC": "YYYY-MM-DDTHH:mm:ss.sssZ"}

    ClientApp->>ClientApp: Store Target Time "ts"
    ClientApp->>ClientApp: Ensure GET /image is cached
    ClientApp->>ClientApp: Schedule Actions for "ts":\n1. Vibrate\n2. Show Loading (3s)\n3. Show Cached Image from /image

    Note over ClientApp: Wait until Target Time "ts"
    ClientApp->>ClientApp: Execute Scheduled Actions

    opt Connection Drop/Reconnect
        ClientApp->>GoServer: Connection Lost
        ClientApp->>ClientApp: Attempt Reconnect
        ClientApp->>GoServer: Connect /ws
        GoServer-->>ClientApp: WebSocket Upgrade OK
        ClientApp->>GoServer: Send {"type": "register", "clientId": "id-123"}
        GoServer-->>ClientApp: Send Current State {"type": "full_state", ...}
        ClientApp->>ClientApp: Re-sync state, ensure image cached, schedule actions if needed.
    end