# Client App Replacement + Core Network Connection Fix

Changes applied:

1. Replaced `client_app/` with the uploaded `client_app.zip` version.
2. Added missing `hzToDb()` helper in `client_app/src/network.js`.
3. Prevented duplicate polling/simulation intervals in `client_app/src/network.js`.
4. Added CORS middleware in `core_network/main.go` so the client app at `http://127.0.0.1:1420` can call `http://localhost:8081/api/v1`.

Run order:

1. `cd core_network && go run .`
2. Open `http://localhost:8081/api/v1/health` and confirm `status: ok`.
3. `cd client_app && npm install && npm run dev`
4. Open `http://127.0.0.1:1420`.
5. The client should show `Core Network Live` when it connects successfully.
