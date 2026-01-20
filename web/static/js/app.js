// Configuration
const API_BASE = '/api/v1';
const REFRESH_INTERVAL = 5000; // 5 seconds

// State
let state = {
    beads: [],
    agents: [],
    projects: [],
    personas: [],
    decisions: [],
    providers: [],
    systemStatus: null
};

let uiState = {
    bead: {
        search: '',
        sort: 'priority',
        priority: 'all',
        type: 'all',
        assigned: '',
        tag: ''
    },
    agent: {
        search: ''
    },
    project: {
        selectedId: ''
    }
};

let busy = new Set();

let modalState = {
    activeId: null,
    lastFocused: null
};

let eventStreamConnected = false;
let reloadTimers = {};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initUI();
    loadAll();
    startEventStream();
    startAutoRefresh();
});

function initUI() {
    const beadSearch = document.getElementById('bead-search');
    const beadSort = document.getElementById('bead-sort');
    const beadPriority = document.getElementById('bead-priority');
    const beadType = document.getElementById('bead-type');
    const beadAssigned = document.getElementById('bead-assigned');
    const beadTag = document.getElementById('bead-tag');
    const beadClear = document.getElementById('bead-clear-filters');

    beadSearch?.addEventListener('input', (e) => {
        uiState.bead.search = e.target.value || '';
        render();
    });
    beadSort?.addEventListener('change', (e) => {
        uiState.bead.sort = e.target.value;
        render();
    });
    beadPriority?.addEventListener('change', (e) => {
        uiState.bead.priority = e.target.value;
        render();
    });
    beadType?.addEventListener('change', (e) => {
        uiState.bead.type = e.target.value;
        render();
    });
    beadAssigned?.addEventListener('input', (e) => {
        uiState.bead.assigned = e.target.value || '';
        render();
    });
    beadTag?.addEventListener('input', (e) => {
        uiState.bead.tag = e.target.value || '';
        render();
    });

    beadClear?.addEventListener('click', () => {
        uiState.bead = {
            search: '',
            sort: 'priority',
            priority: 'all',
            type: 'all',
            assigned: '',
            tag: ''
        };

        if (beadSearch) beadSearch.value = '';
        if (beadSort) beadSort.value = 'priority';
        if (beadPriority) beadPriority.value = 'all';
        if (beadType) beadType.value = 'all';
        if (beadAssigned) beadAssigned.value = '';
        if (beadTag) beadTag.value = '';

        render();
    });

    const agentSearch = document.getElementById('agent-search');
    agentSearch?.addEventListener('input', (e) => {
        uiState.agent.search = e.target.value || '';
        render();
    });

    setupNavActiveState();


    const projectSelect = document.getElementById('project-view-select');
    projectSelect?.addEventListener('change', (e) => {
        uiState.project.selectedId = e.target.value || '';
        render();
    });

    const replSend = document.getElementById('repl-send');
    replSend?.addEventListener('click', () => {
        sendReplQuery();
    });
}

function setupNavActiveState() {
    const links = Array.from(document.querySelectorAll('.nav-list a'));
    const sectionIds = links
        .map((a) => (a.getAttribute('href') || '').replace('#', ''))
        .filter(Boolean);

    function setActive(id) {
        for (const a of links) {
            const targetId = (a.getAttribute('href') || '').replace('#', '');
            if (targetId === id) {
                a.setAttribute('aria-current', 'page');
            } else {
                a.removeAttribute('aria-current');
            }
        }
    }

    window.addEventListener('hashchange', () => {
        const id = (location.hash || '').replace('#', '');
        if (id) setActive(id);
    });

    const sections = sectionIds
        .map((id) => document.getElementById(id))
        .filter(Boolean);

    if (sections.length === 0) return;

    const observer = new IntersectionObserver(
        (entries) => {
            const visible = entries
                .filter((e) => e.isIntersecting)
                .sort((a, b) => (b.intersectionRatio || 0) - (a.intersectionRatio || 0));
            if (visible.length > 0) setActive(visible[0].target.id);
        },
        { rootMargin: '-40% 0px -55% 0px', threshold: [0, 0.2, 0.4, 0.6] }
    );

    for (const s of sections) observer.observe(s);
}

// Auto-refresh
function startAutoRefresh() {
    // Event bus is preferred; this interval is a fallback.
    setInterval(() => {
        if (!eventStreamConnected) loadAll();
    }, REFRESH_INTERVAL);
}

// Load all data
async function loadAll() {
    await Promise.all([
        loadBeads(),
        loadProviders(),
        loadAgents(),
        loadProjects(),
        loadPersonas(),
        loadDecisions(),
        loadSystemStatus()
    ]);
    render();
}

// API calls
async function apiCall(endpoint, options = {}) {
    try {
        const response = await fetch(`${API_BASE}${endpoint}`, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        });
        
        if (!response.ok) {
            let message = 'API request failed';
            try {
                const error = await response.json();
                message = error.error || message;
            } catch {
                // ignore
            }
            throw new Error(message);
        }
        
        if (response.status === 204) {
            return null;
        }
        
        return await response.json();
    } catch (error) {
        console.error('API Error:', error);
        showToast(error.message || 'Request failed', 'error');
        throw error;
    }
}

function scheduleReload(kind, delayMs = 150) {
    if (reloadTimers[kind]) return;
    reloadTimers[kind] = window.setTimeout(async () => {
        try {
            if (kind === 'beads') await loadBeads();
            if (kind === 'agents') await loadAgents();
            if (kind === 'projects') await loadProjects();
            if (kind === 'providers') await loadProviders();
            if (kind === 'decisions') await loadDecisions();
            if (kind === 'status') await loadSystemStatus();
            render();
        } catch (e) {
            // Errors are already surfaced via apiCall toasts.
        } finally {
            window.clearTimeout(reloadTimers[kind]);
            delete reloadTimers[kind];
        }
    }, delayMs);
}

function startEventStream() {
    if (typeof EventSource === 'undefined') return;

    try {
        const es = new EventSource(`${API_BASE}/events/stream`);

        es.addEventListener('connected', () => {
            eventStreamConnected = true;
        });

        const map = {
            'bead.created': ['beads', 'status'],
            'bead.assigned': ['beads', 'agents', 'status'],
            'bead.status_change': ['beads', 'status'],
            'bead.completed': ['beads', 'status'],
            'agent.spawned': ['agents', 'projects', 'status'],
            'agent.status_change': ['agents', 'status'],
            'agent.heartbeat': ['agents', 'status'],
            'agent.completed': ['agents', 'status'],
            'decision.created': ['decisions'],
            'decision.resolved': ['decisions'],
            'provider.registered': ['providers'],
            'provider.deleted': ['providers'],
            'provider.updated': ['providers'],
            'project.created': ['projects'],
            'project.updated': ['projects'],
            'project.deleted': ['projects'],
            'config.updated': ['projects', 'providers', 'agents', 'status']
        };

        for (const [eventName, kinds] of Object.entries(map)) {
            es.addEventListener(eventName, () => {
                for (const k of kinds) scheduleReload(k);
            });
        }

        es.onerror = () => {
            eventStreamConnected = false;
            try {
                es.close();
            } catch {
                // ignore
            }
        };
    } catch {
        eventStreamConnected = false;
    }
}

function showToast(message, type = 'info', timeoutMs = 4500) {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    container.appendChild(toast);

    window.setTimeout(() => {
        toast.remove();
    }, timeoutMs);
}

function setBusy(key, isBusy) {
    if (isBusy) busy.add(key);
    else busy.delete(key);
    render();
}

function isBusy(key) {
    return busy.has(key);
}

async function loadBeads() {
    state.beads = await apiCall('/beads');
}

async function loadAgents() {
    state.agents = await apiCall('/agents');
}

async function loadProviders() {
    state.providers = await apiCall('/providers');
}

async function loadProjects() {
    state.projects = await apiCall('/projects');
}

async function loadPersonas() {
    state.personas = await apiCall('/personas');
}

async function loadDecisions() {
    state.decisions = await apiCall('/decisions');
}

async function loadSystemStatus() {
    state.systemStatus = await apiCall('/system/status');
}

// Render functions
function render() {
    renderSystemStatus();
    renderProjectViewer();
    renderKanban();
    renderProviders();
    renderAgents();
    renderProjects();
    renderPersonas();
    renderDecisions();
}

function renderProjectViewer() {
    const select = document.getElementById('project-view-select');
    const details = document.getElementById('project-view-details');
    if (!select || !details) return;

    const projects = state.projects || [];
    if (projects.length === 0) {
        select.innerHTML = '';
        details.innerHTML = renderEmptyState('No projects configured', 'Add a project to start tracking beads and agents.');
        return;
    }

    if (!uiState.project.selectedId) {
        uiState.project.selectedId = projects[0].id;
    }

    select.innerHTML = projects
        .map((p) => `<option value="${escapeHtml(p.id)}" ${p.id === uiState.project.selectedId ? 'selected' : ''}>${escapeHtml(p.name)} (${escapeHtml(p.id)})</option>`)
        .join('');

    const project = projects.find((p) => p.id === uiState.project.selectedId) || projects[0];
    uiState.project.selectedId = project.id;

    // Get agents assigned to this project
    const projectAgents = (state.agents || []).filter((a) => a.project_id === project.id);

    details.innerHTML = `
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem;">
            <div>
                <h4 style="margin-bottom: 0.75rem; color: var(--primary-color);">Project Details</h4>
                <div><strong>ID:</strong> ${escapeHtml(project.id)}</div>
                <div><strong>Status:</strong> ${escapeHtml(project.status || 'open')}</div>
                <div><strong>Repo:</strong> ${escapeHtml(project.git_repo || '')}</div>
                <div><strong>Branch:</strong> ${escapeHtml(project.branch || '')}</div>
                <div><strong>Beads path:</strong> ${escapeHtml(project.beads_path || '')}</div>
                <div style="margin-top: 0.75rem; display: flex; gap: 0.5rem; flex-wrap: wrap;">
                    <button type="button" class="secondary" onclick="showEditProjectModal('${escapeHtml(project.id)}')">Edit</button>
                    <button type="button" class="danger" onclick="deleteProject('${escapeHtml(project.id)}')">Delete</button>
                </div>
            </div>
            <div>
                <h4 style="margin-bottom: 0.75rem; color: var(--primary-color);">Assigned Agents (${projectAgents.length})</h4>
                <div id="project-agents-list" style="max-height: 300px; overflow-y: auto;">
                    ${renderProjectAgentsList(projectAgents, project.id)}
                </div>
                <div style="margin-top: 0.75rem;">
                    <button type="button" class="secondary" onclick="showAddAgentToProjectModal('${escapeHtml(project.id)}')">+ Add Agent</button>
                </div>
            </div>
        </div>
    `;

    const beads = (state.beads || []).filter((b) => b.project_id === project.id);
    const openBeads = beads.filter((b) => b.status === 'open');
    const inProgressBeads = beads.filter((b) => b.status === 'in_progress');
    const closedBeads = beads.filter((b) => b.status === 'closed');

    const openEl = document.getElementById('project-open-beads');
    const ipEl = document.getElementById('project-in-progress-beads');
    const closedEl = document.getElementById('project-closed-beads');
    if (openEl) {
        openEl.innerHTML = openBeads.length ? openBeads.map(renderBeadCard).join('') : renderEmptyState('No open beads', '');
    }
    if (ipEl) {
        ipEl.innerHTML = inProgressBeads.length ? inProgressBeads.map(renderBeadCard).join('') : renderEmptyState('Nothing in progress', '');
    }
    if (closedEl) {
        closedEl.innerHTML = closedBeads.length ? closedBeads.map(renderBeadCard).join('') : renderEmptyState('No closed beads', '');
    }

    // Keep the assignments board for backward compatibility
    const assignmentsEl = document.getElementById('project-agent-assignments');
    if (assignmentsEl) {
        assignmentsEl.style.display = 'none';
    }
}

function renderProjectAgentsList(agents, projectId) {
    if (agents.length === 0) {
        return `<div class="empty-state" style="padding: 1rem;"><p>No agents assigned yet.</p><p class="small">Add an agent from the org chart to get started.</p></div>`;
    }

    return agents.map((a) => {
        const bead = a.current_bead ? (state.beads || []).find((b) => b.id === a.current_bead) : null;
        const statusClass = a.status === 'working' ? 'working' : (a.status === 'blocked' ? 'blocked' : 'idle');
        const roleName = extractRoleName(a.persona_name || a.name);
        
        return `
            <div class="agent-assignment-row" style="display: flex; justify-content: space-between; align-items: center; padding: 0.5rem; border-bottom: 1px solid var(--border-color);">
                <div>
                    <strong>${escapeHtml(a.name || roleName)}</strong>
                    <span class="badge ${statusClass}" style="margin-left: 0.5rem;">${escapeHtml(a.status || 'idle')}</span>
                    <div class="small" style="color: #6b7280;">
                        ${escapeHtml(a.persona_name || '')}
                        ${bead ? ` • Working on: ${escapeHtml(bead.title.substring(0, 30))}...` : ''}
                    </div>
                </div>
                <button type="button" class="danger" style="padding: 0.25rem 0.5rem; font-size: 0.8rem;" onclick="removeAgentFromProject('${escapeHtml(projectId)}', '${escapeHtml(a.id)}')">Remove</button>
            </div>
        `;
    }).join('');
}

function extractRoleName(personaName) {
    if (!personaName) return 'Agent';
    // Extract role from persona paths like "default/engineering-manager" or "projects/foo/engineering-manager/custom"
    const parts = personaName.split('/');
    let role = parts[parts.length - 1];
    if (parts.length >= 2 && parts[0] === 'default') {
        role = parts[1];
    } else if (parts.length >= 3 && parts[0] === 'projects') {
        role = parts[2];
    }
    // Convert kebab-case to Title Case
    return role.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
}

async function showAddAgentToProjectModal(projectId) {
    // Get available personas (the "org chart")
    const personas = state.personas || [];
    if (personas.length === 0) {
        showToast('No personas available. Add personas to the personas directory first.', 'error');
        return;
    }

    // Get providers
    const providers = state.providers || [];
    if (providers.length === 0) {
        showToast('No providers registered. Register a provider first.', 'error');
        return;
    }

    // Build persona options - only show default personas (the org chart)
    const defaultPersonas = personas.filter(p => p.name && p.name.startsWith('default/'));
    const personaOptions = defaultPersonas.map(p => {
        const roleName = extractRoleName(p.name);
        return { value: p.name, label: `${roleName} (${p.name})` };
    });

    if (personaOptions.length === 0) {
        showToast('No default personas found in org chart. Add personas under personas/default/.', 'error');
        return;
    }

    // Build provider options
    const providerOptions = providers.map(p => ({
        value: p.id,
        label: `${p.name || p.id} (${p.status || 'unknown'})`
    }));

    try {
        const res = await formModal({
            title: 'Add Agent to Project',
            submitText: 'Create & Assign',
            fields: [
                {
                    id: 'persona_name',
                    label: 'Agent Role (from Org Chart)',
                    type: 'select',
                    required: true,
                    options: personaOptions
                },
                {
                    id: 'provider_id',
                    label: 'Provider',
                    type: 'select',
                    required: true,
                    options: providerOptions
                },
                {
                    id: 'custom_name',
                    label: 'Custom Name (optional)',
                    type: 'text',
                    required: false,
                    placeholder: 'Leave empty for default name'
                }
            ]
        });

        if (!res) return;

        // Create the agent and assign to project
        const roleName = extractRoleName(res.persona_name);
        const agentName = res.custom_name || `${roleName} (Default)`;

        setBusy('addAgentToProject', true);
        await apiCall('/agents', {
            method: 'POST',
            body: JSON.stringify({
                name: agentName,
                persona_name: res.persona_name,
                project_id: projectId,
                provider_id: res.provider_id
            })
        });

        showToast(`Agent "${agentName}" created and assigned`, 'success');
        await loadAll();
    } catch (error) {
        // Error already handled by apiCall
    } finally {
        setBusy('addAgentToProject', false);
    }
}

async function removeAgentFromProject(projectId, agentId) {
    const agent = (state.agents || []).find(a => a.id === agentId);
    const agentName = agent ? (agent.name || agentId) : agentId;

    const ok = await confirmModal({
        title: 'Remove Agent?',
        body: `Remove "${agentName}" from this project? This will stop the agent.`,
        confirmText: 'Remove',
        cancelText: 'Cancel',
        danger: true
    });

    if (!ok) return;

    try {
        setBusy(`removeAgent:${agentId}`, true);
        await apiCall(`/agents/${agentId}`, { method: 'DELETE' });
        showToast('Agent removed', 'success');
        await loadAll();
    } catch (error) {
        // Error already handled
    } finally {
        setBusy(`removeAgent:${agentId}`, false);
    }
}

async function assignAgentToProject(projectId) {
    try {
        const res = await formModal({
            title: 'Assign agent to project',
            submitText: 'Assign',
            fields: [{ id: 'agent_id', label: 'Agent ID', type: 'text', required: true, placeholder: 'agent-123' }]
        });
        if (!res) return;

        await apiCall(`/projects/${projectId}/agents`, {
            method: 'POST',
            body: JSON.stringify({ agent_id: res.agent_id, action: 'assign' })
        });

        showToast('Agent assigned', 'success');
        loadAll();
    } catch (error) {
        // Error already handled
    }
}

async function unassignAgentFromProject(projectId, agentId) {
    try {
        await apiCall(`/projects/${projectId}/agents`, {
            method: 'POST',
            body: JSON.stringify({ agent_id: agentId, action: 'unassign' })
        });

        showToast('Agent unassigned', 'success');
        loadAll();
    } catch (error) {
        // Error already handled
    }
}

function renderSystemStatus() {
    const el = document.getElementById('system-status');
    if (!el) return;

    const s = state.systemStatus;
    if (!s) {
        el.innerHTML = '';
        return;
    }

    const badge = s.state === 'active' ? `<span class="badge">active</span>` : `<span class="badge">parked</span>`;
    const reason = s.reason ? escapeHtml(s.reason) : '';
    el.innerHTML = `${badge} ${reason}`;
}

function renderKanban() {
    const filtered = getFilteredBeads();
    const openBeads = filtered.filter((b) => b.status === 'open');
    const inProgressBeads = filtered.filter((b) => b.status === 'in_progress');
    const closedBeads = filtered.filter((b) => b.status === 'closed');

    document.getElementById('open-beads').innerHTML =
        openBeads.length > 0
            ? openBeads.map(renderBeadCard).join('')
            : renderEmptyState('No open beads', 'Create a bead via the API or bd CLI, then it will show up here.');
    document.getElementById('in-progress-beads').innerHTML =
        inProgressBeads.length > 0
            ? inProgressBeads.map(renderBeadCard).join('')
            : renderEmptyState('Nothing in progress', 'Claim a bead to move it into progress.');
    document.getElementById('closed-beads').innerHTML =
        closedBeads.length > 0
            ? closedBeads.map(renderBeadCard).join('')
            : renderEmptyState('No closed beads yet', 'Completed beads will appear here.');
}

function renderProviders() {
    const container = document.getElementById('provider-list');
    if (!container) return;

    if (!state.providers || state.providers.length === 0) {
        container.innerHTML = renderEmptyState(
            'No providers registered',
            'Register at least one vLLM/OpenAI-compatible provider to enable agent execution.',
            '<button type="button" onclick="showRegisterProviderModal()">Register Provider</button>'
        );
        return;
    }

    container.innerHTML = state.providers
        .map((p) => {
            const id = escapeHtml(p.id || '');
            const name = escapeHtml(p.name || p.id || '');
            const endpoint = escapeHtml(p.endpoint || '');
            const model = escapeHtml(p.model || '');
            const configuredModel = escapeHtml(p.configured_model || p.model || '');
            const selectedModel = escapeHtml(p.selected_model || p.model || '');
            const selectionReason = escapeHtml(p.selection_reason || '');
            const modelScore = p.model_score ?? null;
            const selectedGpu = escapeHtml(p.selected_gpu || '');
            const status = escapeHtml(p.status || 'unknown');
            const heartbeatLatency = p.last_heartbeat_latency_ms ?? null;
            const heartbeatError = escapeHtml(p.last_heartbeat_error || '');
            const modelsKey = `providerModels:${id}`;
            const deleteKey = `deleteProvider:${id}`;
            const negotiateKey = `providerNegotiate:${id}`;

            return `
                <div class="provider-card">
                    <div class="provider-header">
                        <div><strong>${name}</strong><div class="small">${id}</div></div>
                        <div><span class="badge">${escapeHtml(p.type || '')}</span></div>
                    </div>
                    <div class="small"><strong>Endpoint:</strong> ${endpoint}</div>
                    <div class="small"><strong>Configured model:</strong> ${configuredModel || '<em>unset</em>'}</div>
                    <div class="small"><strong>Selected model:</strong> ${selectedModel || '<em>unset</em>'}</div>
                    <div class="small"><strong>Selection reason:</strong> ${selectionReason || '<em>pending</em>'}</div>
                    <div class="small"><strong>Model score:</strong> ${modelScore !== null ? escapeHtml(modelScore.toFixed(2)) : '<em>n/a</em>'}</div>
                    <div class="small"><strong>Selected GPU:</strong> ${selectedGpu || '<em>n/a</em>'}</div>
                    <div class="small"><strong>Status:</strong> ${status}</div>
                    <div class="small"><strong>Heartbeat latency:</strong> ${heartbeatLatency !== null && heartbeatLatency !== 0 ? `${escapeHtml(String(heartbeatLatency))}ms` : '<em>n/a</em>'}</div>
                    ${heartbeatError ? `<div class="small"><strong>Heartbeat error:</strong> ${heartbeatError}</div>` : ''}
                    <div class="provider-actions">
                        <button type="button" class="secondary" onclick="fetchProviderModels('${id}')" ${isBusy(modelsKey) ? 'disabled' : ''}>${isBusy(modelsKey) ? 'Loading…' : 'Models'}</button>
                        <button type="button" class="secondary" onclick="renegotiateProvider('${id}')" ${isBusy(negotiateKey) ? 'disabled' : ''}>${isBusy(negotiateKey) ? 'Negotiating…' : 'Re-negotiate model'}</button>
                        <button type="button" class="secondary" onclick="deleteProvider('${id}')" ${isBusy(deleteKey) ? 'disabled' : ''}>${isBusy(deleteKey) ? 'Deleting…' : 'Delete'}</button>
                    </div>
                </div>
            `;
        })
        .join('');
}

function getFilteredBeads() {
    const q = (uiState.bead.search || '').trim().toLowerCase();
    const priority = uiState.bead.priority;
    const type = (uiState.bead.type || '').trim();
    const assigned = (uiState.bead.assigned || '').trim().toLowerCase();
    const tag = (uiState.bead.tag || '').trim().toLowerCase();

    const filtered = state.beads.filter((b) => {
        if (priority !== 'all' && String(b.priority) !== priority) return false;
        if (type !== 'all' && (b.type || '') !== type) return false;
        if (assigned && !(b.assigned_to || '').toLowerCase().includes(assigned)) return false;
        if (tag) {
            const tags = Array.isArray(b.tags) ? b.tags : [];
            if (!tags.some((t) => String(t).toLowerCase().includes(tag))) return false;
        }
        if (q) {
            const hay = `${b.id || ''} ${b.title || ''} ${b.description || ''}`.toLowerCase();
            if (!hay.includes(q)) return false;
        }
        return true;
    });

    const sort = uiState.bead.sort;
    filtered.sort((a, b) => {
        if (sort === 'title') return String(a.title || '').localeCompare(String(b.title || ''));
        if (sort === 'updated_at') {
            return String(b.updated_at || '').localeCompare(String(a.updated_at || ''));
        }
        // priority default
        return (a.priority ?? 99) - (b.priority ?? 99);
    });

    return filtered;
}

function renderEmptyState(title, description, actionsHtml = '') {
    return `
        <div class="empty-state" role="note">
            <h4>${escapeHtml(title)}</h4>
            <p>${escapeHtml(description)}</p>
            ${actionsHtml}
        </div>
    `;
}

function renderBeadCard(bead) {
    const priorityClass = `priority-${bead.priority}`;
    const typeClass = bead.type === 'decision' ? 'decision' : '';

    return `
        <button type="button" class="bead-card ${priorityClass} ${typeClass}" onclick="viewBead('${bead.id}')" aria-label="View bead: ${escapeHtml(bead.title)}">
            <div class="bead-title">${escapeHtml(bead.title)}</div>
            <div class="bead-meta">
                <span class="badge priority-${bead.priority}">P${bead.priority}</span>
                <span class="badge">${escapeHtml(bead.type)}</span>
                ${bead.assigned_to ? `<span class="badge">👤 ${escapeHtml(bead.assigned_to.substring(0, 8))}</span>` : '<span class="badge">unassigned</span>'}
            </div>
        </button>
    `;
}

function renderAgents() {
    const q = (uiState.agent.search || '').trim().toLowerCase();
    const agents = q
        ? state.agents.filter((a) => {
              const hay = `${a.name || ''} ${a.persona_name || ''}`.toLowerCase();
              return hay.includes(q);
          })
        : state.agents;

    const html = agents.map(agent => {
        const statusClass = agent.status;
        return `
            <div class="agent-card ${statusClass}">
                <div class="agent-header">
                    <span class="agent-name">${escapeHtml(agent.name)}</span>
                    <span class="agent-status ${statusClass}">${agent.status}</span>
                </div>
                <div>
                    <strong>Persona:</strong> ${escapeHtml(agent.persona_name)}<br>
                    <strong>Project:</strong> ${agent.project_id.substring(0, 12)}...<br>
                    ${agent.current_bead ? `<strong>Working on:</strong> ${agent.current_bead}` : ''}
                </div>
                <div style="margin-top: 1rem;">
                    <button class="secondary" onclick="cloneAgentPersona('${agent.id}')" ${isBusy(`cloneAgent:${agent.id}`) ? 'disabled' : ''}>${isBusy(`cloneAgent:${agent.id}`) ? 'Cloning…' : 'Clone Persona'}</button>
                    <button class="danger" onclick="stopAgent('${agent.id}')" ${isBusy(`stopAgent:${agent.id}`) ? 'disabled' : ''}>${isBusy(`stopAgent:${agent.id}`) ? 'Stopping…' : 'Stop Agent'}</button>
                </div>
            </div>
        `;
    }).join('');

    document.getElementById('agent-list').innerHTML =
        agents.length > 0
            ? html
            : renderEmptyState(
                  'No active agents',
                  'Spawn an agent to start working on beads.',
                  '<button type="button" class="secondary" onclick="showSpawnAgentModal()">Spawn your first agent</button>'
              );
}

function renderProjects() {
    const html = state.projects.map(project => `
        <div class="project-card">
            <h3>${escapeHtml(project.name)}</h3>
            <div>
                <strong>Branch:</strong> ${escapeHtml(project.branch)}<br>
                <strong>Repo:</strong> ${escapeHtml(project.git_repo)}<br>
                <strong>Agents:</strong> ${project.agents ? project.agents.length : 0}
            </div>
            <div style="margin-top: 0.75rem; display: flex; gap: 0.5rem; flex-wrap: wrap;">
                <button type="button" class="secondary" onclick="viewProject('${escapeHtml(project.id)}')">View</button>
                <button type="button" class="secondary" onclick="showEditProjectModal('${escapeHtml(project.id)}')">Edit</button>
                <button type="button" class="danger" onclick="deleteProject('${escapeHtml(project.id)}')">Delete</button>
            </div>
        </div>
    `).join('');
    
    document.getElementById('project-list').innerHTML =
        html || renderEmptyState('No projects configured', 'Add a project to get started.', '<button type="button" onclick="showCreateProjectModal()">Add Project</button>');
}

function viewProject(projectId) {
    uiState.project.selectedId = projectId;
    location.hash = '#project-viewer';
    render();
}

function projectFormFields(project = {}) {
    return [
        { id: 'name', label: 'Name', type: 'text', required: true, value: project.name || '' },
        { id: 'git_repo', label: 'Git repo', type: 'text', required: true, value: project.git_repo || '' },
        { id: 'branch', label: 'Branch', type: 'text', required: true, value: project.branch || 'main' },
        { id: 'beads_path', label: 'Beads path', type: 'text', required: false, value: project.beads_path || '.beads' },
        {
            id: 'is_perpetual',
            label: 'Perpetual project',
            type: 'select',
            required: false,
            value: project.is_perpetual ? 'true' : 'false',
            options: [
                { value: 'false', label: 'No' },
                { value: 'true', label: 'Yes' }
            ]
        },
        {
            id: 'is_sticky',
            label: 'Sticky project',
            type: 'select',
            required: false,
            value: project.is_sticky ? 'true' : 'false',
            options: [
                { value: 'false', label: 'No' },
                { value: 'true', label: 'Yes' }
            ]
        }
    ];
}

function parseBool(value) {
    return value === 'true' || value === '1' || value === 'yes';
}

function buildProjectPayload(data) {
    const payload = {
        name: (data.name || '').trim(),
        git_repo: (data.git_repo || '').trim(),
        branch: (data.branch || '').trim(),
        beads_path: (data.beads_path || '').trim(),
        is_perpetual: parseBool(data.is_perpetual || 'false'),
        is_sticky: parseBool(data.is_sticky || 'false')
    };

    if (!payload.name) delete payload.name;
    if (!payload.git_repo) delete payload.git_repo;
    if (!payload.branch) delete payload.branch;
    if (!payload.beads_path) delete payload.beads_path;

    return payload;
}

async function showCreateProjectModal() {
    try {
        const res = await formModal({
            title: 'Add project',
            submitText: 'Create',
            fields: projectFormFields()
        });
        if (!res) return;

        await apiCall('/projects', {
            method: 'POST',
            body: JSON.stringify(buildProjectPayload(res))
        });

        showToast('Project created', 'success');
        await loadProjects();
        render();
    } catch (e) {
        // handled
    }
}

async function showEditProjectModal(projectId) {
    const project = state.projects.find((p) => p.id === projectId);
    if (!project) return;

    try {
        const res = await formModal({
            title: `Edit project ${project.name}`,
            submitText: 'Save',
            fields: projectFormFields(project)
        });
        if (!res) return;

        await apiCall(`/projects/${projectId}`, {
            method: 'PUT',
            body: JSON.stringify(buildProjectPayload(res))
        });

        showToast('Project updated', 'success');
        await loadProjects();
        render();
    } catch (e) {
        // handled
    }
}

async function deleteProject(projectId) {
    const project = state.projects.find((p) => p.id === projectId);
    const ok = await confirmModal({
        title: 'Delete project?',
        body: `This will delete project ${project ? project.name : projectId}.`,
        confirmText: 'Delete',
        cancelText: 'Cancel',
        danger: true
    });
    if (!ok) return;

    try {
        await apiCall(`/projects/${projectId}`, { method: 'DELETE' });
        showToast('Project deleted', 'success');
        await loadProjects();
        if (uiState.project.selectedId === projectId) {
            uiState.project.selectedId = '';
        }
        render();
    } catch (e) {
        // handled
    }
}

async function sendReplQuery() {
    const input = document.getElementById('repl-input');
    const responseEl = document.getElementById('repl-response');
    const sendBtn = document.getElementById('repl-send');
    if (!input || !responseEl || !sendBtn) return;

    const message = (input.value || '').trim();
    if (!message) {
        showToast('Enter a question first.', 'error');
        return;
    }

    try {
        setBusy('repl', true);
        sendBtn.disabled = true;
        sendBtn.textContent = 'Sending…';
        responseEl.textContent = 'Sending request…';

        const res = await apiCall('/repl', {
            method: 'POST',
            body: JSON.stringify({ message })
        });

        responseEl.textContent = `${res.response || ''}`.trim() || 'No response returned.';
        if (res.provider_id) {
            responseEl.textContent = `${responseEl.textContent}\n\n— Provider: ${res.provider_name || res.provider_id} (${res.model || 'unknown'})${res.latency_ms ? `, ${res.latency_ms}ms` : ''}`;
        }
    } catch (e) {
        responseEl.textContent = 'Request failed.';
    } finally {
        setBusy('repl', false);
        sendBtn.disabled = false;
        sendBtn.textContent = 'Send';
    }
}

function renderPersonas() {
    const html = state.personas.map(persona => `
        <button type="button" class="persona-card" onclick="editPersona('${escapeHtml(persona.name)}')" aria-label="Edit persona: ${escapeHtml(persona.name)}">
            <h3>🎭 ${escapeHtml(persona.name)}</h3>
            <div>
                <strong>Autonomy:</strong> ${escapeHtml(persona.autonomy_level || 'N/A')}<br>
                <strong>Character:</strong> ${escapeHtml((persona.character || '').substring(0, 100))}...
            </div>
        </button>
    `).join('');
    
    document.getElementById('persona-list').innerHTML =
        html || renderEmptyState('No personas available', 'Add personas under ./personas to populate this list.');
}

async function cloneAgentPersona(agentId) {
    const agent = state.agents.find((a) => a.id === agentId);
    if (!agent) return;

    try {
        const res = await formModal({
            title: 'Clone agent persona',
            submitText: 'Clone',
            fields: [
                { id: 'new_persona_name', label: 'New persona name', type: 'text', required: true, placeholder: 'custom-qa-engineer' },
                { id: 'new_agent_name', label: 'New agent name (optional)', type: 'text', required: false, placeholder: `${agent.name}-custom` },
                { id: 'source_persona', label: 'Source persona (optional)', type: 'text', required: false, placeholder: 'default/qa-engineer' }
            ]
        });
        if (!res) return;

        const replace = await confirmModal({
            title: 'Replace current agent?',
            body: 'Replace this agent with the cloned persona? (Recommended to avoid duplicates.)',
            confirmText: 'Replace',
            cancelText: 'Keep both'
        });

        setBusy(`cloneAgent:${agentId}`, true);
        await apiCall(`/agents/${agentId}/clone`, {
            method: 'POST',
            body: JSON.stringify({
                new_persona_name: res.new_persona_name,
                new_agent_name: res.new_agent_name || '',
                source_persona: res.source_persona || '',
                replace: replace
            })
        });

        showToast('Persona cloned', 'success');
        loadAll();
    } catch (error) {
        // Error already handled
    } finally {
        setBusy(`cloneAgent:${agentId}`, false);
    }
}

function renderDecisions() {
    const html = state.decisions.map(decision => {
        const p0Class = decision.priority === 0 ? 'p0' : '';
        const claimKey = `claimDecision:${decision.id}`;
        const decideKey = `makeDecision:${decision.id}`;
        return `
            <div class="decision-card ${p0Class}">
                <div class="decision-question">${escapeHtml(decision.question)}</div>
                <div>
                    <strong>Priority:</strong> P${decision.priority}<br>
                    <strong>Requester:</strong> ${decision.requester_id}<br>
                    ${decision.recommendation ? `<strong>Recommendation:</strong> ${escapeHtml(decision.recommendation)}` : ''}
                </div>
                <div class="decision-actions">
                    <button class="secondary" onclick="viewDecision('${decision.id}')">View</button>
                    <button onclick="claimDecision('${decision.id}')" ${isBusy(claimKey) ? 'disabled' : ''}>${isBusy(claimKey) ? 'Claiming…' : 'Claim'}</button>
                    ${decision.status === 'in_progress' ? `<button class="secondary" onclick="makeDecision('${decision.id}')" ${isBusy(decideKey) ? 'disabled' : ''}>${isBusy(decideKey) ? 'Submitting…' : 'Decide'}</button>` : ''}
                </div>
            </div>
        `;
    }).join('');

    document.getElementById('decision-list').innerHTML =
        html ||
        renderEmptyState('No pending decisions', 'Decision beads requiring input will appear here.');
}

function viewDecision(decisionId) {
    const d = state.decisions.find((x) => x.id === decisionId);
    if (!d) return;

    const body = `
        <div>
            <div style="margin-bottom: 0.5rem;"><span class="badge priority-${d.priority}">P${d.priority}</span> <span class="badge">decision</span> <span class="badge">${escapeHtml(d.status || '')}</span></div>
            <div><strong>ID:</strong> ${escapeHtml(d.id)}</div>
            <div><strong>Requester:</strong> ${escapeHtml(d.requester_id || '')}</div>
            ${d.recommendation ? `<div style="margin-top: 0.5rem;"><strong>Recommendation:</strong> ${escapeHtml(d.recommendation)}</div>` : ''}
            ${Array.isArray(d.options) && d.options.length > 0 ? `<div style="margin-top: 0.5rem;"><strong>Options:</strong> ${d.options.map((o) => `<span class="badge">${escapeHtml(String(o))}</span>`).join(' ')}</div>` : ''}
            <div style="margin-top: 1rem; white-space: pre-wrap;">${escapeHtml(d.question || '')}</div>
        </div>
    `;

    openAppModal({
        title: 'Decision details',
        bodyHtml: body,
        actions: [
            { label: 'Close', variant: 'secondary', onClick: () => closeAppModal() },
            {
                label: 'Claim',
                onClick: async () => {
                    closeAppModal();
                    await claimDecision(decisionId);
                }
            }
        ]
    });
}

// Actions
function showRegisterProviderModal(preset = {}) {
    formModal({
        title: 'Register provider',
        submitText: 'Register',
        fields: [
            { id: 'id', label: 'Provider ID (e.g. puck)', required: true, placeholder: preset.id || '' },
            { id: 'name', label: 'Display name (optional)', required: false, placeholder: preset.name || '' },
            { id: 'type', label: 'Type (local/openai)', required: false, placeholder: preset.type || 'local' },
            { id: 'endpoint', label: 'Endpoint base (e.g. http://puck.local:8000)', required: true, placeholder: preset.endpoint || '' },
            { id: 'model', label: 'Default model', required: false, placeholder: preset.model || 'NVIDIA-Nemotron-3-Nano-30B-A3B-BF16' }
        ]
    }).then(async (values) => {
        if (!values) return;
        const payload = {
            id: (values.id || '').trim(),
            name: (values.name || '').trim(),
            type: (values.type || '').trim(),
            endpoint: (values.endpoint || '').trim(),
            model: (values.model || '').trim()
        };

        try {
            setBusy('registerProvider', true);
            await apiCall('/providers', { method: 'POST', body: JSON.stringify(payload) });
            showToast('Provider registered', 'success');
            await loadProviders();
            render();
        } finally {
            setBusy('registerProvider', false);
        }
    });
}

async function bootstrapProviders() {
    const ok = await confirmModal({
        title: 'Bootstrap providers?',
        body: 'This will register puck.local and sparky.local using vLLM default port 8000 and model NVIDIA-Nemotron-3-Nano-30B-A3B-BF16.',
        confirmText: 'Bootstrap',
        cancelText: 'Cancel'
    });
    if (!ok) return;

    const presets = [
        { id: 'puck', name: 'puck.local', type: 'local', endpoint: 'http://puck.local:8000', model: 'NVIDIA-Nemotron-3-Nano-30B-A3B-BF16' },
        { id: 'sparky', name: 'sparky.local', type: 'local', endpoint: 'http://sparky.local:8000', model: 'NVIDIA-Nemotron-3-Nano-30B-A3B-BF16' }
    ];

    for (const p of presets) {
        try {
            await apiCall('/providers', { method: 'POST', body: JSON.stringify(p) });
        } catch (e) {
            // ignore individual failures
        }
    }

    await loadProviders();
    showToast('Bootstrap attempted', 'success');
    render();
}

async function fetchProviderModels(providerId) {
    try {
        setBusy(`providerModels:${providerId}`, true);
        const resp = await apiCall(`/providers/${providerId}/models`);
        const models = resp?.models || [];
        const body = models.length > 0
            ? `<div>${models.map((m) => `<div class="badge">${escapeHtml(m.id || '')}</div>`).join(' ')}</div>`
            : '<p>No models returned.</p>';

        openAppModal({
            title: `Models: ${providerId}`,
            bodyHtml: body,
            actions: [{ label: 'Close', variant: 'secondary', onClick: () => closeAppModal() }]
        });
    } catch (e) {
        // handled
    } finally {
        setBusy(`providerModels:${providerId}`, false);
    }
}

async function renegotiateProvider(providerId) {
    try {
        setBusy(`providerNegotiate:${providerId}`, true);
        await apiCall(`/providers/${providerId}/negotiate`, { method: 'POST' });
        showToast('Provider negotiation complete', 'success');
        await loadProviders();
        render();
    } catch (e) {
        // handled
    } finally {
        setBusy(`providerNegotiate:${providerId}`, false);
    }
}

async function deleteProvider(providerId) {
    const ok = await confirmModal({
        title: 'Delete provider?',
        body: `This will remove provider ${providerId}.`,
        confirmText: 'Delete',
        cancelText: 'Cancel',
        danger: true
    });
    if (!ok) return;

    try {
        setBusy(`deleteProvider:${providerId}`, true);
        await apiCall(`/providers/${providerId}`, { method: 'DELETE' });
        showToast('Provider deleted', 'success');
        await loadProviders();
        render();
    } catch (e) {
        // handled
    } finally {
        setBusy(`deleteProvider:${providerId}`, false);
    }
}

function showSpawnAgentModal() {
    // Populate persona and project dropdowns
    const personaSelect = document.getElementById('agent-persona');
    const projectSelect = document.getElementById('agent-project');
    const providerSelect = document.getElementById('agent-provider');
    
    personaSelect.innerHTML = state.personas.map(p => 
        `<option value="${escapeHtml(p.name)}">${escapeHtml(p.name)}</option>`
    ).join('');
    
    projectSelect.innerHTML = state.projects.map(p => 
        `<option value="${p.id}">${escapeHtml(p.name)}</option>`
    ).join('');

    providerSelect.innerHTML = state.providers.map(p =>
        `<option value="${escapeHtml(p.id)}">${escapeHtml(p.name || p.id)}</option>`
    ).join('');
    
    openModal('spawn-agent-modal', { initialFocusSelector: '#agent-name' });
}

function closeSpawnAgentModal() {
    closeModal('spawn-agent-modal');
}

document.getElementById('spawn-agent-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const data = {
        name: formData.get('name'),
        persona_name: formData.get('persona_name'),
        project_id: formData.get('project_id'),
        provider_id: formData.get('provider_id')
    };
    
    try {
        setBusy('spawnAgent', true);

        const submitBtn = e.target.querySelector('button[type="submit"]');
        const prevText = submitBtn?.textContent;
        if (submitBtn) {
            submitBtn.disabled = true;
            submitBtn.textContent = 'Spawning…';
        }

        await apiCall('/agents', {
            method: 'POST',
            body: JSON.stringify(data)
        });

        showToast('Agent spawned', 'success');
        closeSpawnAgentModal();
        loadAll();
    } catch (error) {
        // Error already handled
    } finally {
        const submitBtn = e.target.querySelector('button[type="submit"]');
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.textContent = submitBtn.textContent === 'Spawning…' ? 'Spawn Agent' : submitBtn.textContent;
        }
        setBusy('spawnAgent', false);
    }
});

async function stopAgent(agentId) {
    const ok = await confirmModal({
        title: 'Stop agent?',
        body: 'This will stop the agent and release its file locks.',
        confirmText: 'Stop agent',
        cancelText: 'Cancel',
        danger: true
    });
    if (!ok) return;
    
    try {
        setBusy(`stopAgent:${agentId}`, true);
        await apiCall(`/agents/${agentId}`, {
            method: 'DELETE'
        });

        showToast('Agent stopped', 'success');
        loadAll();
    } catch (error) {
        // Error already handled
    } finally {
        setBusy(`stopAgent:${agentId}`, false);
    }
}

function viewBead(beadId) {
    const bead = state.beads.find(b => b.id === beadId);
    if (!bead) return;

    const tags = Array.isArray(bead.tags) && bead.tags.length > 0 ? bead.tags.map((t) => `<span class="badge">${escapeHtml(String(t))}</span>`).join(' ') : '<em>none</em>';
    const assigned = bead.assigned_to ? escapeHtml(bead.assigned_to) : '<em>unassigned</em>';
    const body = `
        <div>
            <div style="margin-bottom: 0.5rem;"><span class="badge priority-${bead.priority}">P${bead.priority}</span> <span class="badge">${escapeHtml(bead.type)}</span> <span class="badge">${escapeHtml(bead.status)}</span></div>
            <div><strong>ID:</strong> ${escapeHtml(bead.id)}</div>
            <div><strong>Assigned to:</strong> ${assigned}</div>
            <div style="margin-top: 0.5rem;"><strong>Tags:</strong> ${tags}</div>
            <div style="margin-top: 1rem; white-space: pre-wrap;">${escapeHtml(bead.description || 'No description')}</div>
        </div>
    `;

    openAppModal({
        title: bead.title,
        bodyHtml: body,
        actions: [
            { label: 'Redispatch', variant: 'secondary', onClick: () => redispatchBead(bead.id) },
            { label: 'Escalate to CEO', variant: 'secondary', onClick: () => escalateBead(bead.id) },
            { label: 'Close', variant: 'secondary', onClick: () => closeAppModal() }
        ]
    });
}

async function redispatchBead(beadId) {
    try {
        const res = await formModal({
            title: 'Request redispatch',
            submitText: 'Request',
            fields: [{ id: 'reason', label: 'Reason (optional)', type: 'textarea', required: false, placeholder: 'Why should this bead be rerun?' }]
        });
        if (!res) return;

        setBusy(`redispatchBead:${beadId}`, true);
        await apiCall(`/beads/${beadId}/redispatch`, {
            method: 'POST',
            body: JSON.stringify({ reason: res.reason || '' })
        });

        showToast('Redispatch requested', 'success');
        closeAppModal();
        loadAll();
    } catch (error) {
        // Error already handled
    } finally {
        setBusy(`redispatchBead:${beadId}`, false);
    }
}

async function escalateBead(beadId) {
    try {
        const res = await formModal({
            title: 'Escalate to CEO',
            submitText: 'Escalate',
            fields: [
                { id: 'reason', label: 'Decision needed / reason', type: 'textarea', required: true, placeholder: 'What decision is required?' },
                { id: 'returned_to', label: 'Return to (agent/user id, optional)', type: 'text', required: false, placeholder: 'agent-123 or user-jordan' }
            ]
        });
        if (!res) return;

        setBusy(`escalateBead:${beadId}`, true);
        await apiCall(`/beads/${beadId}/escalate`, {
            method: 'POST',
            body: JSON.stringify({ reason: res.reason, returned_to: res.returned_to || '' })
        });

        showToast('Escalated to CEO (decision created)', 'success');
        closeAppModal();
        loadAll();
    } catch (error) {
        // Error already handled
    } finally {
        setBusy(`escalateBead:${beadId}`, false);
    }
}

async function claimDecision(decisionId) {
    try {
        const res = await formModal({
            title: 'Claim decision',
            submitText: 'Claim',
            fields: [
                {
                    id: 'agent_id',
                    label: 'Your user ID (or agent ID)',
                    type: 'text',
                    required: true,
                    placeholder: 'user-jordan or agent-123'
                }
            ]
        });
        if (!res) return;

        setBusy(`claimDecision:${decisionId}`, true);
        await apiCall(`/beads/${decisionId}/claim`, {
            method: 'POST',
            body: JSON.stringify({ agent_id: res.agent_id })
        });

        showToast('Decision claimed', 'success');
        loadAll();
    } catch (error) {
        // Error already handled
    } finally {
        setBusy(`claimDecision:${decisionId}`, false);
    }
}

async function makeDecision(decisionId) {
    try {
        const res = await formModal({
            title: 'Make decision',
            submitText: 'Submit decision',
            fields: [
                { id: 'decision', label: 'Decision', type: 'text', required: true, placeholder: 'APPROVE / DENY / ...' },
                { id: 'rationale', label: 'Rationale', type: 'textarea', required: true, placeholder: 'Why?' },
                { id: 'decider_id', label: 'Your user ID', type: 'text', required: true, placeholder: 'user-jordan' }
            ]
        });
        if (!res) return;

        setBusy(`makeDecision:${decisionId}`, true);
        await apiCall(`/decisions/${decisionId}/decide`, {
            method: 'POST',
            body: JSON.stringify({
                decider_id: res.decider_id,
                decision: res.decision,
                rationale: res.rationale
            })
        });

        showToast('Decision submitted', 'success');
        loadAll();
    } catch (error) {
        // Error already handled
    } finally {
        setBusy(`makeDecision:${decisionId}`, false);
    }
}

function editPersona(personaName) {
    openAppModal({
        title: 'Persona editor (coming soon)',
        bodyHtml: `<p>For now, edit <code>${escapeHtml(personaName)}/PERSONA.md</code> and <code>${escapeHtml(personaName)}/AI_START_HERE.md</code> directly in the repo.</p>`,
        actions: [{ label: 'Close', variant: 'secondary', onClick: () => closeAppModal() }]
    });
}

function closePersonaModal() {
    closeModal('persona-modal');
}

// Utilities
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Close modals when clicking outside
window.onclick = function(event) {
    const spawnModal = document.getElementById('spawn-agent-modal');
    const personaModal = document.getElementById('persona-modal');
    const appModal = document.getElementById('app-modal');
    
    if (event.target === spawnModal) {
        closeSpawnAgentModal();
    }
    if (event.target === personaModal) {
        closePersonaModal();
    }
    if (event.target === appModal) {
        closeAppModal();
    }
}

function getFocusableElements(root) {
    const selector = [
        'a[href]',
        'button:not([disabled])',
        'input:not([disabled])',
        'select:not([disabled])',
        'textarea:not([disabled])',
        '[tabindex]:not([tabindex="-1"])'
    ].join(',');

    return Array.from(root.querySelectorAll(selector)).filter(el => {
        // Skip elements that are hidden via display:none.
        return el.offsetParent !== null || el === document.activeElement;
    });
}

function openModal(modalId, options = {}) {
    const modal = document.getElementById(modalId);
    if (!modal) return;

    modalState.lastFocused = document.activeElement;
    modalState.activeId = modalId;

    modal.style.display = 'block';
    modal.setAttribute('aria-hidden', 'false');
    document.body.style.overflow = 'hidden';

    const initial = options.initialFocusSelector ? modal.querySelector(options.initialFocusSelector) : null;
    (initial || modal).focus();
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;

    modal.style.display = 'none';
    modal.setAttribute('aria-hidden', 'true');
    document.body.style.overflow = '';

    modalState.activeId = null;

    if (modalState.lastFocused && typeof modalState.lastFocused.focus === 'function') {
        modalState.lastFocused.focus();
    }
    modalState.lastFocused = null;
}

document.addEventListener('keydown', (event) => {
    if (!modalState.activeId) return;

    const modal = document.getElementById(modalState.activeId);
    if (!modal) return;

    if (event.key === 'Escape') {
        event.preventDefault();
        closeModal(modalState.activeId);
        return;
    }

    if (event.key !== 'Tab') return;

    const focusables = getFocusableElements(modal);
    if (focusables.length === 0) {
        event.preventDefault();
        return;
    }

    const first = focusables[0];
    const last = focusables[focusables.length - 1];

    if (event.shiftKey) {
        if (document.activeElement === first || document.activeElement === modal) {
            event.preventDefault();
            last.focus();
        }
    } else {
        if (document.activeElement === last) {
            event.preventDefault();
            first.focus();
        }
    }
});

function closeAppModal() {
    closeModal('app-modal');
}

function openAppModal({ title, bodyHtml, actions = [] }) {
    const titleEl = document.getElementById('app-modal-title');
    const bodyEl = document.getElementById('app-modal-body');
    const actionsEl = document.getElementById('app-modal-actions');
    if (!titleEl || !bodyEl || !actionsEl) return;

    titleEl.textContent = title || 'Dialog';
    bodyEl.innerHTML = bodyHtml || '';
    actionsEl.innerHTML = '';

    for (const a of actions) {
        const btn = document.createElement('button');
        btn.type = 'button';
        if (a.variant) btn.className = a.variant;
        btn.textContent = a.label;
        btn.addEventListener('click', a.onClick);
        actionsEl.appendChild(btn);
    }

    openModal('app-modal');
}

function confirmModal({ title, body, confirmText = 'Confirm', cancelText = 'Cancel', danger = false }) {
    return new Promise((resolve) => {
        openAppModal({
            title,
            bodyHtml: `<p>${escapeHtml(body || '')}</p>`,
            actions: [
                {
                    label: cancelText,
                    variant: 'secondary',
                    onClick: () => {
                        closeAppModal();
                        resolve(false);
                    }
                },
                {
                    label: confirmText,
                    variant: danger ? 'danger' : '',
                    onClick: () => {
                        closeAppModal();
                        resolve(true);
                    }
                }
            ]
        });
    });
}

function formModal({ title, submitText = 'Submit', cancelText = 'Cancel', fields = [] }) {
    return new Promise((resolve) => {
        const formId = `modal-form-${Math.random().toString(16).slice(2)}`;
        const bodyHtml = `
            <form id="${formId}">
                ${fields
                    .map((f) => {
                        const id = `field-${formId}-${f.id}`;
                        const required = f.required ? 'required' : '';
                        const placeholder = f.placeholder ? `placeholder="${escapeHtml(f.placeholder)}"` : '';
                        const value = f.value !== undefined && f.value !== null ? String(f.value) : '';
                        if (f.type === 'textarea') {
                            return `
                                <label for="${id}">${escapeHtml(f.label)}</label>
                                <textarea id="${id}" name="${escapeHtml(f.id)}" ${required} ${placeholder}>${escapeHtml(value)}</textarea>
                            `;
                        }
                        if (f.type === 'select') {
                            const options = Array.isArray(f.options) ? f.options : [];
                            return `
                                <label for="${id}">${escapeHtml(f.label)}</label>
                                <select id="${id}" name="${escapeHtml(f.id)}" ${required}>
                                    ${options
                                        .map((opt) => {
                                            const optValue = String(opt.value ?? '');
                                            const selected = optValue === value ? 'selected' : '';
                                            return `<option value="${escapeHtml(optValue)}" ${selected}>${escapeHtml(opt.label ?? optValue)}</option>`;
                                        })
                                        .join('')}
                                </select>
                            `;
                        }
                        return `
                            <label for="${id}">${escapeHtml(f.label)}</label>
                            <input type="text" id="${id}" name="${escapeHtml(f.id)}" ${required} ${placeholder} value="${escapeHtml(value)}">
                        `;
                    })
                    .join('')}
            </form>
        `;

        openAppModal({
            title,
            bodyHtml,
            actions: [
                {
                    label: cancelText,
                    variant: 'secondary',
                    onClick: () => {
                        closeAppModal();
                        resolve(null);
                    }
                },
                {
                    label: submitText,
                    onClick: () => {
                        const form = document.getElementById(formId);
                        if (!form) return;
                        if (!form.reportValidity()) return;
                        const data = new FormData(form);
                        const out = {};
                        for (const [k, v] of data.entries()) out[k] = String(v);
                        closeAppModal();
                        resolve(out);
                    }
                }
            ]
        });

        // focus first field
        window.setTimeout(() => {
            const form = document.getElementById(formId);
            const first = form?.querySelector('input, textarea, select');
            if (first) first.focus();
        }, 0);
    });
}
