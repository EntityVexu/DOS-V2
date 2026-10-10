const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const config = require('./config.json');

// Detect compiled Go binary (Docker) or fallback to `go run` (local dev)
const GO_BINARY = path.join(__dirname, 'vexu-worker');
const HAS_BINARY = fs.existsSync(GO_BINARY);

let state = {
    running: false,
    url: config.targetUrl,
    mode: config.mode || 'get',
    threads: config.threads || 10,
    rps: config.requestsPerSecond || 50,
    startedAt: null,
    requestsSent: 0,
    errors: 0,
    restarts: 0,
    lastRestartAt: null,
    cycleDuration: config.durationSeconds || 60,
    avgLatencyMs: 0
};

let ioRef = null;
let goProc = null;
let restartTimer = null;

function broadcast() {
    if (ioRef) ioRef.emit('status', getStatus());
}

function updateFromGoWorker(payload) {
    if (typeof payload.success === 'number') state.requestsSent = payload.success;
    if (typeof payload.failed === 'number') state.errors = payload.failed;
    if (typeof payload.avgLatencyMs === 'number') state.avgLatencyMs = payload.avgLatencyMs;
    broadcast();
}

function spawnGoWorker() {
    const args = HAS_BINARY ? [] : ['run', 'vexu1.go'];
    const cmd = HAS_BINARY ? GO_BINARY : 'go';

    console.log(`[go-worker] launching: ${cmd} ${args.join(' ')}`);

    goProc = spawn(cmd, args, {
        cwd: __dirname,
        stdio: ['pipe', 'pipe', 'pipe']
    });

    // Send config as first line of stdin
    try {
        goProc.stdin.write(JSON.stringify(config) + '\n');
        goProc.stdin.end();
    } catch (e) {
        console.error('[go-worker] stdin write error:', e.message);
    }

    goProc.stdout.on('data', (d) => {
        const lines = d.toString().split('\n').filter(Boolean);
        for (const line of lines) {
            try {
                updateFromGoWorker(JSON.parse(line));
            } catch (_) {}
        }
    });

    goProc.stderr.on('data', (d) => {
        console.error('[go-worker]', d.toString().trim());
    });

    goProc.on('error', (err) => {
        console.error('[go-worker] spawn error:', err.message);
        state.running = false;
        broadcast();
    });

    goProc.on('exit', (code) => {
        console.log(`[go-worker] exited with code ${code}`);
        state.running = false;
        broadcast();

        if (restartTimer !== null) {
            state.restarts++;
            state.lastRestartAt = Date.now();
            state.startedAt = Date.now();
            state.requestsSent = 0;
            state.errors = 0;
            broadcast();

            restartTimer = setTimeout(() => {
                spawnGoWorker();
                state.running = true;
                broadcast();
            }, 1000);
        }
    });
}

function startWorker(io) {
    ioRef = io;
    state.running = true;
    state.startedAt = Date.now();
    state.requestsSent = 0;
    state.errors = 0;
    state.restarts = 0;
    state.lastRestartAt = null;
    broadcast();
    spawnGoWorker();
    restartTimer = 0;
}

function getStatus() {
    const now = Date.now();
    const uptimeSec = state.startedAt ? Math.floor((now - state.startedAt) / 1000) : 0;
    return {
        running: state.running,
        url: state.url,
        mode: state.mode,
        threads: state.threads,
        rps: state.rps,
        uptimeSec,
        cycleDuration: state.cycleDuration,
        requestsSent: state.requestsSent,
        errors: state.errors,
        avgLatencyMs: state.avgLatencyMs,
        restarts: state.restarts,
        lastRestartAt: state.lastRestartAt,
        startedAt: state.startedAt,
        owner: 'Entity Vexu',
        contact: '@UsAdminChat_bot'
    };
}

module.exports = { startWorker, getStatus, updateFromGoWorker };