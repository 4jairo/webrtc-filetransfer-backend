# Webrtc Filetransfer Backend

Simple HTTP API for interacting with a MongoDB database that stores the WebRTC signaling and files metadata.


Frontend: <https://github.com/4jairo/webrtc-filetransfer-frontend>

- **Clone this Repository**

```bash
git clone https://github.com/4jairo/webrtc-filetransfer-backend.git
cd webrtc-filetransfer-backend
```

- **Install Dependencies**: If you're using Go modules, dependencies will be managed automatically.

```bash
go mod tidy
```

- **Set Up MongoDB**: Make sure the replica set instance of MongoDB is running and accessible. You can update the MongoDB connection URI in `db/mongoClient.go`


- **Run the server**: The server should be listening requests on `http://localhost:8900`
```bash
go run main.go

# or 

docker compose up -d # with automatic MongoDB instance 
```
