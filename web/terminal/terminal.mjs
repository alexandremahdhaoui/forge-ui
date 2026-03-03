import { Terminal } from './xterm.mjs';
import { FitAddon } from './xterm.mjs';

async function init() {
    const container = document.getElementById('terminal-container');

    // 1. Eagerly load terminal WASM in background.
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(
        fetch('/terminal.wasm'),
        go.importObject
    );
    go.run(result.instance);

    // 2. Wait for forgeTerminal to appear (bounded to 10s).
    const maxWaitMs = 10000;
    const pollMs = 50;
    let waited = 0;
    while (!window.forgeTerminal) {
        if (waited >= maxWaitMs) {
            throw new Error('forgeTerminal not available after ' + maxWaitMs + 'ms');
        }
        await new Promise(r => setTimeout(r, pollMs));
        waited += pollMs;
    }

    // 3. Derive workspace and endpoint.
    const params = new URLSearchParams(window.location.search);
    const workspace = container.dataset.workspace || params.get('workspace') || 'default';
    const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const autoEndpoint = wsProto + '//' + window.location.host + '/ws/' + workspace;
    const endpoint = container.dataset.endpoint || params.get('endpoint') || autoEndpoint;

    // 4. Defer xterm creation + SSH session to first panel open.
    let term = null;
    let fitAddon = null;
    let started = false;

    function openTerminal() {
        if (term) return;
        term = new Terminal({ cursorBlink: true, fontSize: 14 });
        fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        term.open(container);
        fitAddon.fit();
        window.addEventListener('resize', () => { if (fitAddon) fitAddon.fit(); });
    }

    function startSession() {
        if (started) return;
        started = true;
        window.forgeTerminal.start(term, workspace, endpoint);
    }

    // 5. Listen for toggle-terminal event from dashboard.
    document.addEventListener('toggle-terminal', () => {
        const panel = document.getElementById('terminal-panel');
        const bar = document.getElementById('terminal-bar');
        if (!panel) return;

        const isOpen = panel.classList.contains('terminal-panel--open');
        panel.classList.toggle('terminal-panel--open');
        if (bar) {
            bar.setAttribute('aria-expanded', String(!isOpen));
        }
        if (!isOpen) {
            openTerminal();
            requestAnimationFrame(() => {
                if (fitAddon) fitAddon.fit();
                startSession();
            });
        }
    });

    // 6. Cleanup on unload.
    window.addEventListener('beforeunload', () => {
        if (window.forgeTerminal) {
            window.forgeTerminal.stop();
        }
    });
}

init().catch(err => console.error('Terminal init failed:', err));
