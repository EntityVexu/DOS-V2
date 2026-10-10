const express = require('express');
const http = require('http');
const { Server } = require('socket.io');
const path = require('path');
const { startWorker, getStatus, updateFromGoWorker } = require('./worker');

const app = express();
const server = http.createServer(app);
const io = new Server(server, {
    cors: { origin: '*' }
});

app.use(express.json());
app.use(express.static(path.join(__dirname, 'public')));

// Read-only status API
app.get('/api/status', (req, res) => {
    res.json(getStatus());
});

// Health check for Render
app.get('/healthz', (req, res) => {
    res.sendStatus(200);
});

// Go worker posts stats here (if you later switch to HTTP push)
app.post('/api/worker-stats', (req, res) => {
    updateFromGoWorker(req.body || {});
    res.sendStatus(200);
});

io.on('connection', (socket) => {
    socket.emit('status', getStatus());
});

const PORT = process.env.PORT || 3000;
server.listen(PORT, '0.0.0.0', () => {
    console.log(`[+] Vexu Dashboard v2 running on port ${PORT}`);
    startWorker(io);
});