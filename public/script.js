const socket = io();
const $ = (id) => document.getElementById(id);

let currentStatus = null;
let prevRequests = 0;
let prevReqTime = Date.now();

function fmtTime(sec) {
    if (sec < 60) return sec + 's';
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    if (h > 0) return `${h}h ${m}m ${s}s`;
    return `${m}m ${s}s`;
}

function fmtNumber(n) {
    return Number(n || 0).toLocaleString('en-US');
}

function log(msg, color = '#7d8590') {
    const el = document.createElement('div');
    el.style.color = color;
    const t = new Date().toLocaleTimeString('en-US', { hour12: false });
    el.textContent = `[${t}] ${msg}`;
    const logBox = $('log');
    logBox.prepend(el);
    if (logBox.children.length > 150) logBox.removeChild(logBox.lastChild);
}

function render(s) {
    if (!s) return;

    const pill = $('statusPill');
    if (s.running) {
        pill.classList.add('running');
        $('statusText').textContent = 'Active';
        $('runVal').textContent = 'Running';
        $('runVal').className = 'stat-value green';
    } else {
        pill.classList.remove('running');
        $('statusText').textContent = 'Idle';
        $('runVal').textContent = 'Stopped';
        $('runVal').className = 'stat-value red';
    }

    $('targetUrl').textContent = s.url || '—';
    $('modeVal').textContent = (s.mode || 'get').toUpperCase();
    $('threadsVal').textContent = s.threads || 10;
    $('rpsVal').textContent = s.rps || 50;
    $('cycleVal').textContent = s.cycleDuration || 60;
    $('uptimeVal').textContent = fmtTime(s.uptimeSec || 0);
    $('reqVal').textContent = fmtNumber(s.requestsSent || 0);
    $('errVal').textContent = fmtNumber(s.errors || 0);
    $('restartVal').textContent = s.restarts || 0;
    $('latVal').textContent = (s.avgLatencyMs || 0) + 'ms';

    const now = Date.now();
    const deltaTime = (now - prevReqTime) / 1000;
    const deltaReq = (s.requestsSent || 0) - prevRequests;
    if (deltaTime > 0.5) {
        const rps = (deltaReq / deltaTime).toFixed(1);
        $('rpsNowVal').textContent = rps;
        prevRequests = s.requestsSent || 0;
        prevReqTime = now;
    }
}

setInterval(() => {
    if (currentStatus && currentStatus.running && currentStatus.startedAt) {
        const uptime = Math.floor((Date.now() - currentStatus.startedAt) / 1000);
        $('uptimeVal').textContent = fmtTime(uptime);
    }
}, 1000);

socket.on('status', (s) => {
    const prev = currentStatus;
    currentStatus = s;
    render(s);

    if (!prev) {
        log(`Connected. Target: ${s.url}`, '#00d4ff');
        if (s.running) log('Worker is live.', '#10b981');
        return;
    }

    if (prev.running !== s.running) {
        log(s.running ? `Worker started → ${s.url}` : 'Worker stopped',
            s.running ? '#10b981' : '#ef4444');
    }

    if (s.restarts > prev.restarts) {
        log(`Cycle complete. Restart #${s.restarts}`, '#f59e0b');
    }

    if (s.requestsSent > prev.requestsSent) {
        const diff = s.requestsSent - prev.requestsSent;
        if (s.requestsSent % 100 === 0 || diff >= 25) {
            log(`Requests sent: ${fmtNumber(s.requestsSent)}`, '#00d4ff');
        }
    }

    if (s.errors > prev.errors) {
        log(`Error detected (total: ${s.errors})`, '#ef4444');
    }
});

socket.on('disconnect', () => {
    log('Disconnected from server', '#ef4444');
});

log('Initializing dashboard...', '#7c3aed');