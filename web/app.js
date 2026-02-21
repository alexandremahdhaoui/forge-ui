// forge-ui WASM application — loads Go wasip1 module and renders pages.
(function () {
    'use strict';

    let wasmModule = null;
    let currentTheme = localStorage.getItem('forge-ui-theme') || 'light';
    let currentSort = 'time';

    // Apply saved theme on load
    if (currentTheme === 'dark') {
        document.documentElement.setAttribute('data-theme', 'dark');
    }

    // ========================================
    // Demo data — realistic forge-ui state
    // ========================================

    const demoData = {
        portfolios: {
            Portfolios: [
                {
                    Name: 'infrastructure',
                    Path: '/home/user/workspaces/infrastructure',
                    IsDefault: false,
                    Workspaces: [
                        {
                            Name: 'platform',
                            Path: '/home/user/workspaces/infrastructure/platform',
                            RepoCount: 3,
                            Repos: [
                                { Name: 'forge', WorkspaceName: 'platform', Path: '/home/user/workspaces/infrastructure/platform/forge', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: '#', LastCommitTime: '2026-02-21T10:00:00Z' },
                                { Name: 'forge-ui', WorkspaceName: 'platform', Path: '/home/user/workspaces/infrastructure/platform/forge-ui', Branch: 'feat/wasm', IsDirty: true, Ahead: 3, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: '#', LastCommitTime: '2026-02-21T12:30:00Z' },
                                { Name: 'config-server', WorkspaceName: 'platform', Path: '/home/user/workspaces/infrastructure/platform/config-server', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 2, HasUpstream: true, HasForge: true, RepoLink: '#', LastCommitTime: '2026-02-20T08:15:00Z' },
                            ],
                            AllStages: ['lint', 'unit', 'integration'],
                            RepoForge: [
                                { RepoName: 'forge', RepoLink: '#', StageResults: { lint: 'passed', unit: 'passed', integration: 'passed' } },
                                { RepoName: 'forge-ui', RepoLink: '#', StageResults: { lint: 'passed', unit: 'passed', integration: '' } },
                                { RepoName: 'config-server', RepoLink: '#', StageResults: { lint: 'passed', unit: 'failed', integration: '' } },
                            ],
                        },
                        {
                            Name: 'networking',
                            Path: '/home/user/workspaces/infrastructure/networking',
                            RepoCount: 2,
                            Repos: [
                                { Name: 'mesh-proxy', WorkspaceName: 'networking', Path: '/home/user/workspaces/infrastructure/networking/mesh-proxy', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: '#', LastCommitTime: '2026-02-19T14:00:00Z' },
                                { Name: 'dns-controller', WorkspaceName: 'networking', Path: '/home/user/workspaces/infrastructure/networking/dns-controller', Branch: 'develop', IsDirty: true, Ahead: 1, Behind: 0, HasUpstream: true, HasForge: false, RepoLink: '#', LastCommitTime: '2026-02-21T09:00:00Z' },
                            ],
                            AllStages: ['lint', 'unit'],
                            RepoForge: [
                                { RepoName: 'mesh-proxy', RepoLink: '#', StageResults: { lint: 'passed', unit: 'passed' } },
                            ],
                        },
                    ],
                    Stats: { TotalWorkspaces: 2, TotalRepos: 5, DirtyRepos: 2, TotalTests: 42, Passed: 38, Failed: 4 },
                },
                {
                    Name: 'default',
                    Path: '/home/user/workspaces',
                    IsDefault: true,
                    Workspaces: [
                        {
                            Name: 'personal',
                            Path: '/home/user/workspaces/personal',
                            RepoCount: 2,
                            Repos: [
                                { Name: 'dotfiles', WorkspaceName: 'personal', Path: '/home/user/workspaces/personal/dotfiles', Branch: 'main', IsDirty: true, Ahead: 5, Behind: 0, HasUpstream: true, HasForge: false, RepoLink: '#', LastCommitTime: '2026-02-21T11:00:00Z' },
                                { Name: 'blog', WorkspaceName: 'personal', Path: '/home/user/workspaces/personal/blog', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 0, HasUpstream: true, HasForge: false, RepoLink: '#', LastCommitTime: '2026-02-18T16:00:00Z' },
                            ],
                            AllStages: [],
                            RepoForge: [],
                        },
                    ],
                    Stats: { TotalWorkspaces: 1, TotalRepos: 2, DirtyRepos: 1, TotalTests: 0, Passed: 0, Failed: 0 },
                },
            ],
            Stats: { TotalPortfolios: 2, TotalWorkspaces: 3, TotalRepos: 7, DirtyRepos: 3, TotalTests: 42, Passed: 38, Failed: 4 },
            SortMode: 'time',
            DarkMode: false,
            HomeURL: '#',
        },

        portfolio_infrastructure: {
            Name: 'infrastructure',
            Path: '/home/user/workspaces/infrastructure',
            IsDefault: false,
            Workspaces: [
                {
                    Name: 'platform',
                    Path: '/home/user/workspaces/infrastructure/platform',
                    RepoCount: 3,
                    Repos: [
                        { Name: 'forge', WorkspaceName: 'platform', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: '#', LastCommitTime: '2026-02-21T10:00:00Z' },
                        { Name: 'forge-ui', WorkspaceName: 'platform', Branch: 'feat/wasm', IsDirty: true, Ahead: 3, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: '#', LastCommitTime: '2026-02-21T12:30:00Z' },
                        { Name: 'config-server', WorkspaceName: 'platform', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 2, HasUpstream: true, HasForge: true, RepoLink: '#', LastCommitTime: '2026-02-20T08:15:00Z' },
                    ],
                    AllStages: ['lint', 'unit', 'integration'],
                    RepoForge: [
                        { RepoName: 'forge', RepoLink: '#', StageResults: { lint: 'passed', unit: 'passed', integration: 'passed' } },
                        { RepoName: 'forge-ui', RepoLink: '#', StageResults: { lint: 'passed', unit: 'passed', integration: '' } },
                        { RepoName: 'config-server', RepoLink: '#', StageResults: { lint: 'passed', unit: 'failed', integration: '' } },
                    ],
                },
                {
                    Name: 'networking',
                    Path: '/home/user/workspaces/infrastructure/networking',
                    RepoCount: 2,
                    Repos: [
                        { Name: 'mesh-proxy', WorkspaceName: 'networking', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: '#', LastCommitTime: '2026-02-19T14:00:00Z' },
                        { Name: 'dns-controller', WorkspaceName: 'networking', Branch: 'develop', IsDirty: true, Ahead: 1, Behind: 0, HasUpstream: true, HasForge: false, RepoLink: '#', LastCommitTime: '2026-02-21T09:00:00Z' },
                    ],
                    AllStages: ['lint', 'unit'],
                    RepoForge: [
                        { RepoName: 'mesh-proxy', RepoLink: '#', StageResults: { lint: 'passed', unit: 'passed' } },
                    ],
                },
            ],
            Stats: { TotalWorkspaces: 2, TotalRepos: 5, DirtyRepos: 2, TotalTests: 42, Passed: 38, Failed: 4 },
            SortMode: 'time',
            DarkMode: false,
            HomeURL: '#',
        },

        workspace_platform: {
            Name: 'platform',
            PortfolioName: 'infrastructure',
            Path: '/home/user/workspaces/infrastructure/platform',
            Repos: [
                {
                    Name: 'forge', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 0, HasUpstream: true, HasForge: true,
                    StatusFiles: [], DiffStat: '', RepoLink: '#', LastCommitTime: '2026-02-21T10:00:00Z',
                    RecentLogs: [
                        { Hash: 'a1b2c3d', Message: 'feat: add generic-builder engine' },
                        { Hash: 'e4f5g6h', Message: 'fix: resolve container build caching' },
                        { Hash: 'i7j8k9l', Message: 'docs: update README with WASM support' },
                    ],
                },
                {
                    Name: 'forge-ui', Branch: 'feat/wasm', IsDirty: true, Ahead: 3, Behind: 0, HasUpstream: true, HasForge: true,
                    StatusFiles: [{ Code: 'M', FilePath: 'cmd/forge-ui-wasm/main.go' }, { Code: 'A', FilePath: 'web/index.html' }, { Code: 'A', FilePath: 'web/app.js' }],
                    DiffStat: ' 3 files changed, 450 insertions(+)', RepoLink: '#', LastCommitTime: '2026-02-21T12:30:00Z',
                    RecentLogs: [
                        { Hash: 'x1y2z3a', Message: 'feat: implement WASM rendering engine' },
                        { Hash: 'b4c5d6e', Message: 'feat: add WASI shim for browser' },
                        { Hash: 'f7g8h9i', Message: 'chore: configure forge generic-builder for WASM' },
                    ],
                },
                {
                    Name: 'config-server', Branch: 'main', IsDirty: false, Ahead: 0, Behind: 2, HasUpstream: true, HasForge: true,
                    StatusFiles: [], DiffStat: '', RepoLink: '#', LastCommitTime: '2026-02-20T08:15:00Z',
                    RecentLogs: [
                        { Hash: 'j1k2l3m', Message: 'fix: reload config on SIGHUP' },
                        { Hash: 'n4o5p6q', Message: 'test: add integration tests for hot reload' },
                    ],
                },
            ],
            Stats: { TotalRepos: 3, ForgeRepos: 3, TotalTests: 30, Passed: 27, Failed: 3, Skipped: 2 },
            AllStages: ['lint', 'unit', 'integration'],
            RepoForge: [
                { RepoName: 'forge', RepoLink: '#', StageResults: { lint: 'passed', unit: 'passed', integration: 'passed' } },
                { RepoName: 'forge-ui', RepoLink: '#', StageResults: { lint: 'passed', unit: 'passed', integration: '' } },
                { RepoName: 'config-server', RepoLink: '#', StageResults: { lint: 'passed', unit: 'failed', integration: '' } },
            ],
            SortMode: 'time',
            DarkMode: false,
            HomeURL: '#',
        },

        forge_forge: {
            WorkspaceName: 'platform',
            RepoName: 'forge',
            PortfolioName: 'infrastructure',
            Spec: {
                Name: 'forge',
                Build: [
                    { Name: 'forge', Src: './cmd/forge', Dest: './build/bin', Engine: 'go://go-build' },
                    { Name: 'forge-container', Src: './containers/forge/Containerfile', Engine: 'go://container-build' },
                ],
                Test: [
                    { Name: 'lint', Testenv: '', Runner: 'go://go-lint' },
                    { Name: 'unit', Testenv: '', Runner: 'go://go-test' },
                    { Name: 'integration', Testenv: 'kind-cluster', Runner: 'go://go-test' },
                ],
            },
            Artifacts: [
                { Name: 'forge', Type: 'binary', Location: './build/bin/forge', Timestamp: '2026-02-21T10:00:00Z', Version: 'a1b2c3d' },
                { Name: 'forge-container', Type: 'container', Location: 'ghcr.io/user/forge:latest', Timestamp: '2026-02-21T10:05:00Z', Version: 'a1b2c3d' },
            ],
            TestReports: [
                { ID: 'rpt-001', Stage: 'lint', Status: 'passed', StartTime: '2026-02-21T09:50:00Z', Duration: 4.2, Stats: { Total: 1, Passed: 1, Failed: 0, Skipped: 0 }, Coverage: { Enabled: false, Percentage: 0, FilePath: '' }, ErrorMessage: '' },
                { ID: 'rpt-002', Stage: 'unit', Status: 'passed', StartTime: '2026-02-21T09:51:00Z', Duration: 12.8, Stats: { Total: 87, Passed: 85, Failed: 0, Skipped: 2 }, Coverage: { Enabled: true, Percentage: 78.4, FilePath: 'coverage.out' }, ErrorMessage: '' },
                { ID: 'rpt-003', Stage: 'integration', Status: 'passed', StartTime: '2026-02-21T09:55:00Z', Duration: 45.1, Stats: { Total: 12, Passed: 12, Failed: 0, Skipped: 0 }, Coverage: { Enabled: false, Percentage: 0, FilePath: '' }, ErrorMessage: '' },
            ],
            TestEnvs: [
                { ID: 'env-001', Name: 'kind-cluster', Status: 'passed', CreatedAt: '2026-02-21T09:54:00Z', UpdatedAt: '2026-02-21T10:01:00Z', ManagedResources: ['kind-cluster/forge-test', 'namespace/forge-integration'] },
            ],
            Stats: { TotalTests: 100, Passed: 98, Failed: 0, Skipped: 2, AvgCoverage: 78.4, HasCoverage: true, StageCount: 3 },
            StageStatusMap: { lint: 'passed', unit: 'passed', integration: 'passed' },
            DarkMode: false,
            HomeURL: '#',
        },
    };

    // ========================================
    // WASM execution
    // ========================================

    async function loadWasmModule() {
        const response = await fetch('forge-ui.wasm');
        wasmModule = await WebAssembly.compile(await response.arrayBuffer());
    }

    async function runWasm(command) {
        const encoder = new TextEncoder();
        const stdin = encoder.encode(JSON.stringify(command));

        const wasi = new WASIShim({ stdin: stdin });
        const importObject = wasi.getImportObject();
        const instance = await WebAssembly.instantiate(wasmModule, importObject);

        const exitCode = wasi.start(instance);
        const stdout = wasi.getStdout();
        const stderr = wasi.getStderr();

        if (exitCode !== 0) {
            console.error('WASM stderr:', stderr);
            throw new Error(`WASM exited with code ${exitCode}: ${stderr}`);
        }

        return stdout;
    }

    // ========================================
    // Navigation and rendering
    // ========================================

    function getDataForPage(page, params) {
        switch (page) {
            case 'portfolios':
                return demoData.portfolios;
            case 'portfolio':
                return demoData['portfolio_' + params.name] || demoData.portfolio_infrastructure;
            case 'workspace':
                return demoData['workspace_' + params.workspace] || demoData.workspace_platform;
            case 'forge':
                return demoData['forge_' + params.repo] || demoData.forge_forge;
            default:
                return demoData.portfolios;
        }
    }

    async function renderPage(page, params) {
        params = params || {};
        const sort = params.sort || currentSort;
        currentSort = sort;

        const data = getDataForPage(page, params);

        const command = {
            action: 'render',
            page: page,
            theme: currentTheme,
            sort: sort,
            data: data,
        };

        try {
            const html = await runWasm(command);
            document.getElementById('content').innerHTML = html;
        } catch (e) {
            console.error('Render error:', e);
            document.getElementById('content').innerHTML =
                '<div class="empty-state body-large">Error rendering page: ' + e.message + '</div>';
        }
    }

    // ========================================
    // Public API (used by template onclick handlers)
    // ========================================

    window.forgeUI = {
        navigate: function (page, params) {
            renderPage(page, params);
        },

        toggleTheme: function () {
            currentTheme = currentTheme === 'light' ? 'dark' : 'light';
            localStorage.setItem('forge-ui-theme', currentTheme);
            if (currentTheme === 'dark') {
                document.documentElement.setAttribute('data-theme', 'dark');
            } else {
                document.documentElement.removeAttribute('data-theme');
            }
        },
    };

    // ========================================
    // Initialization
    // ========================================

    async function init() {
        try {
            await loadWasmModule();
            await renderPage('portfolios');
        } catch (e) {
            console.error('Failed to initialize:', e);
            document.getElementById('content').innerHTML =
                '<div class="empty-state body-large">Failed to load WASM module. ' +
                'Make sure forge-ui.wasm is built and available. Error: ' + e.message + '</div>';
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
