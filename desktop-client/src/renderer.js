// AI Agent Desktop - Renderer Process
// Reuse existing web UI features and add Electron desktop client capabilities

const API_BASE = 'http://localhost:3001/api';
let isLoading = false;
let currentStreamController = null;
let messageHistory = [];
const CHAT_MEMORY_ROLES = new Set(['user', 'assistant']);
const runningScheduledTaskIds = new Set();
let selectedImages = [];

// Initialize
document.addEventListener('DOMContentLoaded', async function() {
    // Load configuration
    await loadConfig();

    // Check status
    checkStatus();

    // Setup event listeners
    setupEventListeners();

    // Setup Electron menu event listeners
    setupElectronListeners();

    // Load persisted memory into chat history
    await refreshMemory(false);

    // Initialize scheduled tasks (Electron only)
    if (window.electronAPI) {
        await initScheduledTasks();
    }

    // Focus input field
    document.getElementById('messageInput').focus();
});

// Load configuration
async function loadConfig() {
    try {
        if (window.electronAPI) {
            const config = await window.electronAPI.getConfig();
            if (config.apiBase) {
                document.getElementById('apiBaseInput').value = config.apiBase;
            }
        }
    } catch (error) {
        console.error('Error loading config:', error);
    }
}

// Setup event listeners
function setupEventListeners() {
    // Input field enter to send
    document.getElementById('messageInput').addEventListener('keypress', function(e) {
        if (e.key === 'Enter' && !isLoading) {
            sendMessage();
        }
    });
    
    // Streaming mode indicator
    document.querySelectorAll('input[name="chatMode"]').forEach(radio => {
        radio.addEventListener('change', function() {
            updateStreamIndicator(this.value === 'streaming');
        });
    });

    document.addEventListener('click', async function(event) {
        const viewBtn = event.target.closest('[data-history-action="view-memory"]');
        if (viewBtn) {
            await viewTaskHistoryMemory(viewBtn.dataset.historyId || '');
            return;
        }

        const deleteBtn = event.target.closest('[data-history-action="delete-memory"]');
        if (deleteBtn) {
            await deleteTaskHistoryEntry(deleteBtn.dataset.historyId || '');
        }
    });
}

// Setup Electron menu event listeners
function setupElectronListeners() {
    if (!window.electronAPI) return;
    
    // New chat
    window.electronAPI.onMenuNewChat((event) => {
        clearChat();
    });
    
    // Clear chat
    window.electronAPI.onMenuClearChat((event) => {
        clearChat();
    });
    
    // API settings
    window.electronAPI.onMenuApiSettings((event) => {
        showConfig();
    });
    
    // Refresh config
    window.electronAPI.onMenuRefreshConfig((event) => {
        refreshConfig();
    });
}

// Update streaming mode indicator
function updateStreamIndicator(isStreaming) {
    const indicator = document.querySelector('.streaming-indicator');
    if (indicator) {
        indicator.style.display = isStreaming ? 'inline-block' : 'none';
    }
}

// Check API status
async function checkStatus() {
    try {
        const response = await fetch(`${API_BASE}/agent/status`);
        if (response.ok) {
            const data = await response.json();
            updateStatus(true, 'Connected');
            refreshConfig();
        } else {
            updateStatus(false, 'Connection failed');
        }
    } catch (error) {
        updateStatus(false, 'Disconnected');
        console.error('Status check error:', error);
    }
}

// Update status display
function updateStatus(connected, text) {
    const dot = document.getElementById('statusDot');
    const statusText = document.getElementById('statusText');
    const apiStatus = document.getElementById('api-status');
    
    dot.className = 'status-dot' + (connected ? ' connected' : ' error');
    statusText.textContent = text;
    apiStatus.textContent = 'API: ' + text;
}

function parseMarkdown(content) {
    if (window.marked && typeof window.marked.parse === 'function') {
        return window.marked.parse(content);
    }
    const div = document.createElement('div');
    div.textContent = content;
    return div.innerHTML.replace(/\n/g, '<br>');
}

// Send message
async function sendMessage() {
    const input = document.getElementById('messageInput');
    const message = input.value.trim();
    const mode = document.querySelector('input[name="chatMode"]:checked').value;
    
    if ((!message && selectedImages.length === 0) || isLoading) return;
    
    isLoading = true;
    updateUIState();
    
    // Add user message
    addMessage(formatUserMessage(message, selectedImages.length), 'user');
    input.value = '';
    
    // Save to history
    messageHistory.push({ role: 'user', content: formatUserMessage(message, selectedImages.length) });
    const imagePayload = selectedImages.map(item => item.data);
    
    if (mode === 'streaming') {
        await sendStreamingMessage(message, imagePayload);
    } else {
        await sendBlockingMessage(message, imagePayload);
    }

    clearSelectedImages();
    
    isLoading = false;
    updateUIState();
}

// Send blocking mode message
async function sendBlockingMessage(message, images = []) {
    try {
        const response = await fetch(`${API_BASE}/agent/chat`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                message: message,
                images: images,
                stream: false
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            addMessage(data.response, 'agent');
            messageHistory.push({ role: 'agent', content: data.response });
        } else {
            addMessage('Error: Unable to get AI response', 'agent', true);
        }
    } catch (error) {
        addMessage('Error: ' + error.message, 'agent', true);
        updateStatus(false, 'Connection error');
    }
}

// Send streaming message
async function sendStreamingMessage(message, images = []) {
    try {
        const response = await fetch(`${API_BASE}/agent/chat`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                message: message,
                images: images,
                stream: true
            })
        });
        
        if (!response.ok) {
            addMessage('Error: Unable to start streaming response', 'agent', true);
            return;
        }
        
        // Create streaming message element
        const messageElement = addStreamingMessage();
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        
        currentStreamController = {
            reader: reader,
            cancelled: false
        };
        
        let content = '';
        
        try {
            while (!currentStreamController.cancelled) {
                const { done, value } = await reader.read();
                if (done) break;
                
                const event = decoder.decode(value, { stream: true });
                const lines = event.split('\n');
                let eventType = null;
                
                for (let line of lines) {
                    line = line.trim();
                    if (line.startsWith('event:')) {
                        eventType = line.slice(6).trim();
                        continue;
                    }
                    if (line.startsWith('data:')) {
                        const dataStr = line.slice(5).trim();
                        try {
                            const data = JSON.parse(dataStr);
                            if (eventType === 'message' && data.content) {
                                content += data.content;
                                updateStreamingMessage(messageElement, content);
                            } else if (eventType === 'complete') {
                                finalizeStreamingMessage(messageElement, content);
                                messageHistory.push({ role: 'agent', content: content });
                                currentStreamController = null;
                                return;
                            } else if (eventType === 'error') {
                                finalizeStreamingMessage(messageElement, null, 'Error: ' + (data.error || 'Unknown error'));
                                currentStreamController = null;
                                return;
                            }
                        } catch (e) {
                            console.log('Failed to parse SSE data:', dataStr);
                        }
                    }
                    eventType = null;
                }
            }
        } catch (error) {
            finalizeStreamingMessage(messageElement, null, 'Streaming error: ' + error.message);
        } finally {
            currentStreamController = null;
        }
        
    } catch (error) {
        addMessage('Error: ' + error.message, 'agent', true);
        updateStatus(false, 'Connection error');
    }
}

// Add message to chat area
function addMessage(content, sender, isError = false) {
    const chatMessages = document.getElementById('chatMessages');
    
    // Remove welcome message
    const welcomeMsg = chatMessages.querySelector('.welcome-message');
    if (welcomeMsg) {
        welcomeMsg.remove();
    }
    
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${sender}`;
    
    const headerDiv = document.createElement('div');
    headerDiv.className = 'message-header';
    headerDiv.innerHTML = sender === 'user' 
        ? '<span>Me</span>'
        : '<span class="avatar agent">🤖</span><span>AI Agent</span>';
    
    const bubbleDiv = document.createElement('div');
    bubbleDiv.className = 'message-bubble';
    
    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';
    
    if (isError) {
        contentDiv.style.color = '#dc3545';
        contentDiv.textContent = String(content ?? '');
    } else {
        contentDiv.innerHTML = parseMarkdown(content);
    }
    
    bubbleDiv.appendChild(contentDiv);
    messageDiv.appendChild(headerDiv);
    messageDiv.appendChild(bubbleDiv);
    chatMessages.appendChild(messageDiv);
    
    // Scroll to bottom
    chatMessages.scrollTop = chatMessages.scrollHeight;
    
    return messageDiv;
}

// Add streaming message
function addStreamingMessage() {
    const chatMessages = document.getElementById('chatMessages');
    
    // Remove welcome message
    const welcomeMsg = chatMessages.querySelector('.welcome-message');
    if (welcomeMsg) {
        welcomeMsg.remove();
    }
    
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message agent';
    
    const headerDiv = document.createElement('div');
    headerDiv.className = 'message-header';
    headerDiv.innerHTML = '<span class="avatar agent">🤖</span><span>AI Agent</span><span class="streaming-indicator"></span>';
    
    const bubbleDiv = document.createElement('div');
    bubbleDiv.className = 'message-bubble';
    
    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';
    contentDiv.innerHTML = '<span class="cursor">▋</span>';
    
    bubbleDiv.appendChild(contentDiv);
    messageDiv.appendChild(headerDiv);
    messageDiv.appendChild(bubbleDiv);
    chatMessages.appendChild(messageDiv);
    
    chatMessages.scrollTop = chatMessages.scrollHeight;
    
    return { messageDiv, contentDiv };
}

// Update streaming message
function updateStreamingMessage(messageElement, content) {
    messageElement.contentDiv.innerHTML = parseMarkdown(content) + '<span class="cursor">▋</span>';
    
    const chatMessages = document.getElementById('chatMessages');
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Finalize streaming message
function finalizeStreamingMessage(messageElement, content, error = null) {
    if (error) {
        messageElement.contentDiv.style.color = '#dc3545';
        messageElement.contentDiv.textContent = String(error ?? '');
    } else if (content !== null) {
        messageElement.contentDiv.innerHTML = parseMarkdown(content);
    }
    
    // Remove streaming indicator
    const indicator = messageElement.messageDiv.querySelector('.streaming-indicator');
    if (indicator) {
        indicator.remove();
    }
}

// Update UI state
function updateUIState() {
    const sendBtn = document.getElementById('sendBtn');
    const plannerBtn = document.getElementById('plannerBtn');
    const uploadImageBtn = document.getElementById('uploadImageBtn');
    const input = document.getElementById('messageInput');
    const isTaskRunning = runningScheduledTaskIds.size > 0;
    
    if (isLoading || isTaskRunning) {
        sendBtn.disabled = true;
        sendBtn.innerHTML = isLoading ? '<div class="loading-spinner"></div>' : 'Send';
        plannerBtn.disabled = true;
        uploadImageBtn.disabled = true;
        input.disabled = true;
    } else {
        sendBtn.disabled = false;
        sendBtn.innerHTML = 'Send';
        plannerBtn.disabled = false;
        uploadImageBtn.disabled = false;
        input.disabled = false;
        input.focus();
    }
}

// Clear chat
function clearChat() {
    const chatMessages = document.getElementById('chatMessages');
    chatMessages.innerHTML = `
        <div class="welcome-message">
            <h2>👋 Welcome to AI Agent Desktop</h2>
            <p>This is a powerful AI Agent desktop client. You can chat with AI, use various skills, or let Agent Planner help you plan and execute tasks.</p>
        </div>
    `;
    
    messageHistory = [];
    clearSelectedImages();
    
    // Cancel ongoing stream
    if (currentStreamController) {
        currentStreamController.cancelled = true;
        currentStreamController = null;
    }
}

function formatUserMessage(message, imageCount) {
    if (message && imageCount > 0) {
        return `${message}\n[${imageCount} image(s) uploaded]`;
    }
    if (message) {
        return message;
    }
    return `[${imageCount} image(s) uploaded]`;
}

function updateUploadHint() {
    const uploadHint = document.getElementById('uploadHint');
    if (!uploadHint) {
        return;
    }
    if (!selectedImages.length) {
        uploadHint.textContent = '';
        return;
    }
    uploadHint.textContent = `Selected image(s): ${selectedImages.map(item => item.name).join(', ')}`;
}

function clearSelectedImages() {
    selectedImages = [];
    updateUploadHint();
}

function extractFileName(filePath) {
    const parts = String(filePath || '').split(/[\\/]/).filter(Boolean);
    return parts.length ? parts[parts.length - 1] : 'uploaded-image';
}

async function selectImagesForUpload() {
    if (!window.electronAPI) {
        showNotification('Image upload is only available in desktop app', 'error');
        return;
    }

    const result = await window.electronAPI.showOpenDialog({
        properties: ['openFile', 'multiSelections'],
        filters: [{ name: 'Images', extensions: ['png', 'jpg', 'jpeg', 'webp', 'gif', 'bmp'] }]
    });

    if (result.canceled || !Array.isArray(result.filePaths) || result.filePaths.length === 0) {
        return;
    }

    const uploaded = [];
    const failed = [];
    for (const filePath of result.filePaths) {
        const readResult = await window.electronAPI.readFileAsBase64(filePath);
        if (!readResult || !readResult.success || !readResult.data) {
            failed.push(extractFileName(filePath));
            continue;
        }
        uploaded.push({ name: extractFileName(filePath), data: readResult.data });
    }

    if (failed.length > 0) {
        showNotification(`Failed to read ${failed.length} image(s)`, 'error');
    }

    if (uploaded.length === 0) {
        return;
    }

    selectedImages = uploaded;
    updateUploadHint();
}

// Run Agent Planner
async function runPlanner() {
    const input = document.getElementById('messageInput');
    const task = input.value.trim();
    
    if (!task) {
        input.focus();
        return;
    }
    
    const mode = document.querySelector('input[name="chatMode"]:checked').value;
    const plannerMessage = "Please create a detailed plan for the following task, then execute it step by step. Use MCP skills as much as possible. Task: " + task;
    
    isLoading = true;
    updateUIState();
    
    // Add user message
    addMessage("Execute plan task: " + task, 'user');
    messageHistory.push({ role: 'user', content: task });
    
    if (mode === 'streaming') {
        await sendStreamingMessage(plannerMessage);
    } else {
        await sendBlockingMessage(plannerMessage);
    }
    
    isLoading = false;
    updateUIState();
    input.value = '';
}

// Refresh config
async function refreshConfig() {
    try {
        const response = await fetch(`${API_BASE}/agent/config`);
        if (response.ok) {
            const data = await response.json();
            const configDisplay = document.getElementById('configDisplay');
            configDisplay.innerHTML = `
                <div class="config-item">
                    <label>Chat Model</label>
                    <input type="text" value="${data.chatModel || 'N/A'}" readonly>
                </div>
                <div class="config-item">
                    <label>Embedding Model</label>
                    <input type="text" value="${data.embeddingModel || 'N/A'}" readonly>
                </div>
                <div class="config-item">
                    <label>Mode</label>
                    <input type="text" value="${data.agentMode || 'N/A'}" readonly>
                </div>
                <div class="config-item">
                    <label>Role</label>
                    <input type="text" value="${data.character || 'N/A'}" readonly>
                </div>
            `;
        }
    } catch (error) {
        document.getElementById('configDisplay').innerHTML = 
            '<p style="color: #dc3545;">Failed to load configuration</p>';
    }
}

// Clear memory
async function clearMemory() {
    try {
        const response = await fetch(`${API_BASE}/agent/memory`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            showNotification('Memory cleared', 'success');
            await refreshMemory(false);
        } else {
            showNotification('Failed to clear memory', 'error');
        }
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

// Refresh memory
function normalizeChatContexts(contexts) {
    if (!Array.isArray(contexts)) {
        return [];
    }
    return contexts
        .filter(ctx => ctx && typeof ctx.content === 'string' && typeof ctx.role === 'string' && CHAT_MEMORY_ROLES.has(ctx.role))
        .map(ctx => ({
            role: ctx.role,
            content: ctx.content
        }));
}

function toSender(role) {
    return role === 'user' ? 'user' : 'agent';
}

function renderChatFromMemory(contexts) {
    const chatMessages = document.getElementById('chatMessages');
    if (contexts.length === 0) {
        chatMessages.innerHTML = `
            <div class="welcome-message">
                <h2>👋 Welcome to AI Agent Desktop</h2>
                <p>This is a powerful AI Agent desktop client. You can chat with AI, use various skills, or let Agent Planner help you plan and execute tasks.</p>
            </div>
        `;
        messageHistory = [];
        return;
    }

    chatMessages.innerHTML = '';
    messageHistory = contexts.map(ctx => ({
        role: toSender(ctx.role),
        content: ctx.content
    }));
    contexts.forEach(ctx => {
        const sender = toSender(ctx.role);
        addMessage(ctx.content, sender);
    });
}

async function refreshMemory(showSuccess = true) {
    try {
        const response = await fetch(`${API_BASE}/agent/memory`);
        if (response.ok) {
            const data = await response.json();
            const chatContexts = normalizeChatContexts(data.contexts);
            renderChatFromMemory(chatContexts);
            if (showSuccess) {
                showNotification(`Memory refreshed (${chatContexts.length})`, 'success');
            }
        } else {
            showNotification('Failed to refresh memory', 'error');
        }
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

// Show config panel
function showConfig() {
    document.getElementById('configPanel').classList.add('active');
    refreshConfig();
}

// Show chat area
function showChat() {
    document.getElementById('configPanel').classList.remove('active');
    document.getElementById('scheduledTasksPanel').classList.remove('active');
    document.getElementById('chatArea').style.display = 'flex';
    document.querySelector('.top-bar').style.display = 'flex';
}

// Save API configuration
async function saveApiConfig() {
    const apiBase = document.getElementById('apiBaseInput').value.trim();
    
    if (window.electronAPI) {
        const config = await window.electronAPI.getConfig();
        config.apiBase = apiBase;
        await window.electronAPI.saveConfig(config);
    }
    
    showNotification('Configuration saved', 'success');
    showChat();
    checkStatus();
}

// Export chat history
async function exportChat() {
    if (messageHistory.length === 0) {
        showNotification('No chat history to export', 'error');
        return;
    }
    
    const content = messageHistory.map(m => {
        const role = m.role === 'user' ? 'User' : 'AI Agent';
        return `[${role}]\n${m.content}\n`;
    }).join('\n---\n\n');
    
    if (window.electronAPI) {
        const result = await window.electronAPI.showSaveDialog({
            defaultPath: `chat-export-${new Date().toISOString().slice(0, 10)}.txt`,
            filters: [
                { name: 'Text files', extensions: ['txt'] },
                { name: 'All files', extensions: ['*'] }
            ]
        });
        
        if (!result.canceled && result.filePath) {
            const writeResult = await window.electronAPI.writeFile(result.filePath, content);
            if (writeResult.success) {
                showNotification('Chat history exported', 'success');
            } else {
                showNotification('Export failed: ' + writeResult.error, 'error');
            }
        }
    } else {
        // Browser environment: use download
        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `chat-export-${new Date().toISOString().slice(0, 10)}.txt`;
        a.click();
        URL.revokeObjectURL(url);
    }
}

// Show notification
function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 12px 20px;
        border-radius: 8px;
        font-size: 14px;
        z-index: 10000;
        animation: slideIn 0.3s ease;
        ${type === 'success' ? 'background: #28a745; color: white;' : ''}
        ${type === 'error' ? 'background: #dc3545; color: white;' : ''}
        ${type === 'info' ? 'background: #667eea; color: white;' : ''}
    `;
    notification.textContent = message;
    
    document.body.appendChild(notification);
    
    // Remove after 3 seconds
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Add animation styles
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
    .cursor {
        animation: blink 1s infinite;
    }
    @keyframes blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0; }
    }
`;
document.head.appendChild(style);

// ==========================================
// Scheduled Tasks Functions
// ==========================================

// Initialize scheduled tasks on load
async function initScheduledTasks() {
    if (window.electronAPI) {
        // Setup listener for scheduled task execution events
        window.electronAPI.onScheduledTaskExecuted((data) => {
            showScheduledTaskNotification(data);
            refreshTaskHistory();
        });
        window.electronAPI.onScheduledTaskBeforeExecute(async (data) => {
            await handleScheduledTaskPreparation(data);
        });
        window.electronAPI.onScheduledTaskRunState((data) => {
            handleScheduledTaskRunState(data);
        });

        // Load scheduled tasks
        await loadScheduledTasks();

        // Load task history
        await refreshTaskHistory();
    }
}

// Show scheduled tasks panel
function showScheduledTasks() {
    // Update sidebar navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });
    event.target.closest('.nav-item')?.classList.add('active');

    // Hide other panels
    document.getElementById('configPanel').classList.remove('active');

    // Show scheduled tasks panel
    document.getElementById('scheduledTasksPanel').classList.add('active');

    // Hide chat area
    document.getElementById('chatArea').style.display = 'none';
    document.querySelector('.top-bar').style.display = 'none';

    // Load tasks
    loadScheduledTasks();
    refreshTaskHistory();
}

// Show chat area (called from scheduled tasks panel)
function returnToChat() {
    document.getElementById('scheduledTasksPanel').classList.remove('active');
    document.getElementById('chatArea').style.display = 'flex';
    document.querySelector('.top-bar').style.display = 'flex';

    // Update sidebar
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });
    document.querySelector('.nav-item[onclick="showChat()"]')?.classList.add('active');
}

// Toggle interval input based on schedule type
function toggleIntervalInput() {
    const scheduleType = document.getElementById('taskScheduleType').value;
    const intervalContainer = document.getElementById('intervalContainer');

    if (scheduleType === 'interval') {
        intervalContainer.style.display = 'block';
    } else {
        intervalContainer.style.display = 'none';
    }
}

// Load scheduled tasks
async function loadScheduledTasks() {
    if (!window.electronAPI) return;

    try {
        const tasksData = await window.electronAPI.getScheduledTasks();
        renderScheduledTasks(tasksData.tasks || []);
    } catch (error) {
        console.error('Error loading scheduled tasks:', error);
    }
}

// Render scheduled tasks list
function renderScheduledTasks(tasks) {
    const container = document.getElementById('scheduledTasksList');

    if (!tasks || tasks.length === 0) {
        container.innerHTML = '<p style="color: #888; font-size: 13px;">No scheduled tasks yet</p>';
        return;
    }

    container.innerHTML = tasks.map(task => `
        <div class="task-item" data-task-id="${task.id}">
            <div class="task-item-header">
                <span class="task-item-name">${escapeHtml(task.name)}</span>
                <span class="task-item-status ${task.enabled ? 'enabled' : 'disabled'}">
                    ${task.enabled ? 'Active' : 'Paused'}
                </span>
            </div>
            <div class="task-item-message">${escapeHtml(task.message)}</div>
            <div class="task-item-schedule">
                Schedule: ${getScheduleLabel(task)}
            </div>
            <div class="task-item-actions">
                <button class="task-action-btn ${task.enabled ? 'stop' : 'play'}"
                        onclick="toggleScheduledTask('${task.id}')">
                    ${task.enabled ? 'Pause' : 'Resume'}
                </button>
                <button class="task-action-btn run" onclick="runScheduledTask('${task.id}')">
                    Run Now
                </button>
                <button class="task-action-btn delete" onclick="deleteScheduledTask('${task.id}')">
                    Delete
                </button>
            </div>
        </div>
    `).join('');
}

// Get schedule label
function getScheduleLabel(task) {
    switch (task.scheduleType) {
        case 'interval':
            return `Every ${task.intervalMinutes} minutes`;
        case 'hourly':
            return 'Hourly';
        case 'daily':
            return 'Daily';
        case 'weekly':
            return 'Weekly';
        default:
            return 'Unknown';
    }
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Add new scheduled task
async function addScheduledTask() {
    if (!window.electronAPI) {
        showNotification('Electron API not available', 'error');
        return;
    }

    const name = document.getElementById('taskName').value.trim();
    const message = document.getElementById('taskMessage').value.trim();
    const scheduleType = document.getElementById('taskScheduleType').value;
    const intervalMinutes = parseInt(document.getElementById('taskInterval').value) || 60;
    const executeImmediately = document.getElementById('taskExecuteImmediately').checked;

    if (!name) {
        showNotification('Please enter a task name', 'error');
        return;
    }

    if (!message) {
        showNotification('Please enter a message', 'error');
        return;
    }

    const task = {
        name,
        message,
        scheduleType,
        intervalMinutes,
        executeImmediately,
        enabled: true
    };

    try {
        const result = await window.electronAPI.addScheduledTask(task);
        if (result.success) {
            showNotification('Task created successfully', 'success');
            // Clear form
            document.getElementById('taskName').value = '';
            document.getElementById('taskMessage').value = '';
            document.getElementById('taskInterval').value = '60';
            document.getElementById('taskExecuteImmediately').checked = false;
            // Reload tasks
            await loadScheduledTasks();
        } else {
            showNotification('Failed to create task', 'error');
        }
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

// Toggle scheduled task enabled/disabled
async function toggleScheduledTask(taskId) {
    if (!window.electronAPI) return;

    try {
        const result = await window.electronAPI.toggleScheduledTask(taskId);
        if (result.success) {
            showNotification(`Task ${result.enabled ? 'enabled' : 'disabled'}`, 'info');
            await loadScheduledTasks();
        }
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

// Run scheduled task manually
async function runScheduledTask(taskId) {
    if (!window.electronAPI) return;

    try {
        const result = await window.electronAPI.executeScheduledTask(taskId);
        if (result.success) {
            showNotification('Task executed', 'success');
            await refreshTaskHistory();
        } else {
            showNotification('Error: ' + result.error, 'error');
        }
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

function hasChatHistoryOrDraft() {
    const input = document.getElementById('messageInput');
    const hasDraft = !!(input && input.value.trim());
    const hasHistory = Array.isArray(messageHistory) && messageHistory.length > 0;
    return hasDraft || hasHistory;
}

async function handleScheduledTaskPreparation(data) {
    if (!window.electronAPI || !data || !data.requestId) return;

    const needsReset = hasChatHistoryOrDraft();
    if (!needsReset) {
        await window.electronAPI.resolveScheduledTaskPreparation({
            requestId: data.requestId,
            confirmed: true,
            resetRequired: false
        });
        return;
    }

    const taskName = sanitizeDialogText(data.taskName || '');
    const confirmed = confirm(`检测到当前聊天有历史记录或未发送内容。是否清空聊天并启动定时任务「${taskName}」？`);
    if (confirmed) {
        clearChat();
        const input = document.getElementById('messageInput');
        if (input) {
            input.value = '';
        }
    }

    await window.electronAPI.resolveScheduledTaskPreparation({
        requestId: data.requestId,
        confirmed,
        resetRequired: confirmed
    });
}

function handleScheduledTaskRunState(data) {
    if (!data || typeof data.running !== 'boolean' || !data.taskId) {
        return;
    }

    if (data.running) {
        runningScheduledTaskIds.add(data.taskId);
    } else {
        runningScheduledTaskIds.delete(data.taskId);
    }
    updateUIState();
}

function sanitizeDialogText(text) {
    return String(text).replace(/[\u0000-\u001F\u007F]/g, '').slice(0, 120);
}

// Delete scheduled task
async function deleteScheduledTask(taskId) {
    if (!window.electronAPI) return;

    if (!confirm('Are you sure you want to delete this task?')) {
        return;
    }

    try {
        const result = await window.electronAPI.deleteScheduledTask(taskId);
        if (result.success) {
            showNotification('Task deleted', 'success');
            await loadScheduledTasks();
        }
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

// Refresh task history
async function refreshTaskHistory() {
    if (!window.electronAPI) return;

    try {
        const history = await window.electronAPI.getTaskHistory();
        renderTaskHistory(history);
    } catch (error) {
        console.error('Error loading task history:', error);
    }
}

// Render task history
function renderTaskHistory(history) {
    const container = document.getElementById('taskHistoryList');

    if (!history || history.length === 0) {
        container.innerHTML = '<p style="color: #888; font-size: 13px;">No execution history</p>';
        return;
    }

    container.innerHTML = '';
    history.slice(0, 10).forEach(item => {
        const historyItem = document.createElement('div');
        historyItem.className = `history-item ${item.isError ? 'error' : ''}`;

        const header = document.createElement('div');
        header.className = 'history-item-header';
        const nameSpan = document.createElement('span');
        nameSpan.textContent = item.taskName || item.taskId || '';
        const timeSpan = document.createElement('span');
        timeSpan.textContent = formatDate(item.timestamp);
        header.appendChild(nameSpan);
        header.appendChild(timeSpan);

        const resultDiv = document.createElement('div');
        resultDiv.className = 'history-item-result';
        resultDiv.textContent = item.result || '';

        const actionsDiv = document.createElement('div');
        actionsDiv.className = 'task-item-actions';
        actionsDiv.style.marginTop = '6px';

        const viewBtn = document.createElement('button');
        viewBtn.className = 'task-action-btn run';
        viewBtn.textContent = 'View Memory';
        viewBtn.setAttribute('data-history-action', 'view-memory');
        viewBtn.setAttribute('data-history-id', item.id || '');

        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'task-action-btn delete';
        deleteBtn.textContent = 'Delete Record';
        deleteBtn.setAttribute('data-history-action', 'delete-memory');
        deleteBtn.setAttribute('data-history-id', item.id || '');

        actionsDiv.appendChild(viewBtn);
        actionsDiv.appendChild(deleteBtn);

        historyItem.appendChild(header);
        historyItem.appendChild(resultDiv);
        historyItem.appendChild(actionsDiv);
        container.appendChild(historyItem);
    });
}

// Format date
function formatDate(isoString) {
    if (!isoString) return '';
    const date = new Date(isoString);
    return date.toLocaleString();
}

// Show notification when scheduled task is executed
function showScheduledTaskNotification(data) {
    showNotification(`Task "${data.taskName}" executed`, 'info');
}

function formatTaskMemory(contexts) {
    if (!Array.isArray(contexts) || contexts.length === 0) {
        return 'No memory saved for this task execution.';
    }
    return contexts.map((ctx, index) => {
        const role = typeof ctx?.role === 'string' ? ctx.role : 'unknown';
        const content = typeof ctx?.content === 'string' ? ctx.content : JSON.stringify(ctx);
        return `${index + 1}. [${role}] ${content}`;
    }).join('\n\n');
}

async function viewTaskHistoryMemory(historyId) {
    if (!window.electronAPI || !historyId) return;
    try {
        const history = await window.electronAPI.getTaskHistory();
        const item = Array.isArray(history) ? history.find(entry => entry.id === historyId) : null;
        if (!item) {
            showNotification('History record not found', 'error');
            return;
        }
        showTaskMemoryModal(item.taskName || item.taskId || 'Task Memory', formatTaskMemory(item.contexts));
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

async function deleteTaskHistoryEntry(historyId) {
    if (!window.electronAPI || !historyId) return;
    if (!confirm('Delete this execution record and its local memory?')) {
        return;
    }
    try {
        const result = await window.electronAPI.deleteTaskHistoryEntry(historyId);
        if (result.success) {
            showNotification('Execution record deleted', 'success');
            await refreshTaskHistory();
        } else {
            showNotification('Error: ' + result.error, 'error');
        }
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

function showTaskMemoryModal(title, content) {
    const overlay = document.createElement('div');
    overlay.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,.45);z-index:11000;display:flex;align-items:center;justify-content:center;padding:20px;';

    const modal = document.createElement('div');
    modal.style.cssText = 'width:min(900px,95vw);max-height:85vh;background:#fff;border-radius:10px;display:flex;flex-direction:column;overflow:hidden;';

    const header = document.createElement('div');
    header.style.cssText = 'padding:12px 16px;border-bottom:1px solid #eee;display:flex;justify-content:space-between;align-items:center;';
    const headerTitle = document.createElement('strong');
    headerTitle.textContent = title;
    const closeBtn = document.createElement('button');
    closeBtn.textContent = 'Close';
    closeBtn.className = 'task-action-btn delete';
    closeBtn.onclick = () => overlay.remove();
    header.appendChild(headerTitle);
    header.appendChild(closeBtn);

    const body = document.createElement('pre');
    body.style.cssText = 'margin:0;padding:14px 16px;overflow:auto;white-space:pre-wrap;word-break:break-word;font-size:12px;line-height:1.5;';
    body.textContent = content;

    modal.appendChild(header);
    modal.appendChild(body);
    overlay.appendChild(modal);
    overlay.addEventListener('click', (event) => {
        if (event.target === overlay) {
            overlay.remove();
        }
    });
    document.body.appendChild(overlay);
}
