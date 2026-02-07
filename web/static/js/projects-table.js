// Projects Table with full CRUD

async function renderProjectsTable() {
    const container = document.getElementById('projects-table-container');
    if (!container) return;

    try {
        const projects = await apiCall('/projects');

        if (!projects || projects.length === 0) {
            container.innerHTML = renderEmptyState(
                'No projects configured',
                'Add a project to get started',
                '<button type="button" onclick="showProjectOptionsMenu()">Add Project</button>'
            );
            return;
        }

        const table = `
            <div class="table-container">
                <table class="data-table">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Git Repository</th>
                            <th>Branch</th>
                            <th>Status</th>
                            <th>Type</th>
                            <th>Beads</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${projects.map(project => renderProjectRow(project)).join('')}
                    </tbody>
                </table>
            </div>
        `;

        container.innerHTML = table;
    } catch (error) {
        container.innerHTML = `<div class="error">Failed to load projects: ${error.message}</div>`;
    }
}

function renderProjectRow(project) {
    const statusClass = project.status === 'open' ? 'status-success' :
                       project.status === 'closed' ? 'status-muted' : 'status-warning';

    const beadsPath = project.beads_path || '.beads';
    const typeLabel = project.is_perpetual ? '♾️ Perpetual' :
                     project.is_sticky ? '📌 Sticky' : '📁 Regular';

    return `
        <tr data-project-id="${project.id}">
            <td>
                <strong>${escapeHtml(project.name)}</strong>
                <br><small class="text-muted">${project.id}</small>
            </td>
            <td>
                <code class="code-small">${escapeHtml(project.git_repo)}</code>
            </td>
            <td><span class="badge">${escapeHtml(project.branch)}</span></td>
            <td><span class="status-badge ${statusClass}">${project.status}</span></td>
            <td>${typeLabel}</td>
            <td>
                <span class="badge">${beadsPath}</span>
            </td>
            <td>
                <div class="action-buttons">
                    <button type="button" class="btn-icon" onclick="viewProject('${project.id}')"
                            title="View project">👁️</button>
                    <button type="button" class="btn-icon" onclick="showEditProjectModal('${project.id}')"
                            title="Edit project">✏️</button>
                    <button type="button" class="btn-icon" onclick="showGitOperations('${project.id}')"
                            title="Git operations">🔄</button>
                    ${!project.is_perpetual ? `
                        <button type="button" class="btn-icon btn-danger"
                                onclick="confirmDeleteProject('${project.id}')"
                                title="Delete project">🗑️</button>
                    ` : ''}
                </div>
            </td>
        </tr>
    `;
}

async function showEditProjectModal(projectId) {
    try {
        const project = await apiCall(`/projects/${projectId}`);

        const res = await formModal({
            title: 'Edit Project',
            submitText: 'Save Changes',
            fields: [
                { id: 'name', label: 'Name', type: 'text', required: true, value: project.name },
                { id: 'git_repo', label: 'Git Repository', type: 'text', required: true, value: project.git_repo },
                { id: 'branch', label: 'Branch', type: 'text', required: true, value: project.branch },
                { id: 'beads_path', label: 'Beads Path', type: 'text', value: project.beads_path || '.beads' },
                {
                    id: 'is_perpetual',
                    label: 'Perpetual (never closes)',
                    type: 'select',
                    value: project.is_perpetual ? 'true' : 'false',
                    options: [
                        { value: 'false', label: 'No' },
                        { value: 'true', label: 'Yes' }
                    ]
                },
                {
                    id: 'is_sticky',
                    label: 'Sticky (auto-load)',
                    type: 'select',
                    value: project.is_sticky ? 'true' : 'false',
                    options: [
                        { value: 'false', label: 'No' },
                        { value: 'true', label: 'Yes' }
                    ]
                },
                {
                    id: 'git_strategy',
                    label: 'Git Strategy',
                    type: 'select',
                    value: project.git_strategy || 'direct',
                    options: [
                        { value: 'direct', label: 'Direct to branch' },
                        { value: 'branch-pr', label: 'Feature branch + PR' }
                    ]
                }
            ]
        });

        if (!res) return;

        // Update project
        await apiCall(`/projects/${projectId}`, {
            method: 'PUT',
            body: JSON.stringify({
                name: res.name,
                git_repo: res.git_repo,
                branch: res.branch,
                beads_path: res.beads_path,
                is_perpetual: res.is_perpetual === 'true',
                is_sticky: res.is_sticky === 'true',
                git_strategy: res.git_strategy || 'direct'
            })
        });

        showToast('Project updated successfully', 'success');
        await loadProjects();
        renderProjectsTable();
    } catch (error) {
        showToast(`Failed to update project: ${error.message}`, 'error');
    }
}

async function confirmDeleteProject(projectId) {
    const project = await apiCall(`/projects/${projectId}`);

    const confirmed = confirm(
        `Are you sure you want to delete project "${project.name}"?\n\n` +
        `This will remove the project from Loom but will NOT delete files from disk.\n\n` +
        `This action cannot be undone.`
    );

    if (!confirmed) return;

    try {
        await apiCall(`/projects/${projectId}`, { method: 'DELETE' });
        showToast('Project deleted successfully', 'success');
        await loadProjects();
        renderProjectsTable();
    } catch (error) {
        showToast(`Failed to delete project: ${error.message}`, 'error');
    }
}

async function showGitOperations(projectId) {
    const project = await apiCall(`/projects/${projectId}`);

    const modalHTML = `
        <div class="modal show" id="git-ops-modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>Git Operations - ${escapeHtml(project.name)}</h2>
                    <button type="button" class="modal-close" onclick="closeGitOpsModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="git-ops-container">
                        <div class="git-status" id="git-status-display">
                            <div class="loading">Loading git status...</div>
                        </div>

                        <div class="git-actions">
                            <button type="button" class="primary" onclick="gitSync('${projectId}')">
                                🔄 Sync (Pull)
                            </button>
                            <button type="button" class="secondary" onclick="gitCommit('${projectId}')">
                                💾 Commit Changes
                            </button>
                            <button type="button" class="secondary" onclick="gitPush('${projectId}')">
                                ⬆️ Push to Remote
                            </button>
                            <button type="button" class="secondary" onclick="refreshGitStatus('${projectId}')">
                                🔃 Refresh Status
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;

    document.body.insertAdjacentHTML('beforeend', modalHTML);
    loadGitStatus(projectId);
}

async function loadGitStatus(projectId) {
    const display = document.getElementById('git-status-display');
    try {
        const status = await apiCall(`/projects/git/status?project_id=${projectId}`);

        display.innerHTML = `
            <div class="git-status-info">
                <h3>Repository Status</h3>
                <pre class="code-block">${escapeHtml(status.status || 'No changes')}</pre>

                ${status.branch ? `<p><strong>Branch:</strong> ${escapeHtml(status.branch)}</p>` : ''}
                ${status.ahead ? `<p class="text-warning">⬆️ ${status.ahead} commits ahead</p>` : ''}
                ${status.behind ? `<p class="text-warning">⬇️ ${status.behind} commits behind</p>` : ''}
            </div>
        `;
    } catch (error) {
        display.innerHTML = `<div class="error">Failed to load git status: ${error.message}</div>`;
    }
}

function refreshGitStatus(projectId) {
    loadGitStatus(projectId);
}

async function gitSync(projectId) {
    try {
        showToast('Syncing with remote...', 'info');
        await apiCall('/projects/git/sync', {
            method: 'POST',
            body: JSON.stringify({ project_id: projectId })
        });
        showToast('Git sync completed', 'success');
        loadGitStatus(projectId);
    } catch (error) {
        showToast(`Git sync failed: ${error.message}`, 'error');
    }
}

async function gitCommit(projectId) {
    const message = prompt('Commit message:');
    if (!message) return;

    try {
        showToast('Creating commit...', 'info');
        await apiCall('/projects/git/commit', {
            method: 'POST',
            body: JSON.stringify({
                project_id: projectId,
                message: message
            })
        });
        showToast('Commit created', 'success');
        loadGitStatus(projectId);
    } catch (error) {
        showToast(`Commit failed: ${error.message}`, 'error');
    }
}

async function gitPush(projectId) {
    const confirmed = confirm('Push commits to remote repository?');
    if (!confirmed) return;

    try {
        showToast('Pushing to remote...', 'info');
        await apiCall('/projects/git/push', {
            method: 'POST',
            body: JSON.stringify({ project_id: projectId })
        });
        showToast('Git push completed', 'success');
        loadGitStatus(projectId);
    } catch (error) {
        showToast(`Git push failed: ${error.message}`, 'error');
    }
}

function closeGitOpsModal() {
    const modal = document.getElementById('git-ops-modal');
    if (modal) modal.remove();
}
