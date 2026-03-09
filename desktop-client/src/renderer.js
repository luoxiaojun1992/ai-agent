// AI Agent Desktop - Renderer Process
// Reuse existing web UI features and add Electron desktop client capabilities

const API_BASE = 'http://localhost:3001/api';
let isLoading = false;
let currentStreamController = null;
let messageHistory = [];

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

// Send message
async function sendMessage() {
    const input = document.getElementById('messageInput');
    const message = input.value.trim();
    const mode = document.querySelector('input[name="chatMode"]:checked').value;
    
    if (!message || isLoading) return;
    
    isLoading = true;
    updateUIState();
    
    // Add user message
    addMessage(message, 'user');
    input.value = '';
    
    // Save to history
    messageHistory.push({ role: 'user', content: message });
    
    if (mode === 'streaming') {
        await sendStreamingMessage(message);
    } else {
        await sendBlockingMessage(message);
    }
    
    isLoading = false;
    updateUIState();
}

// Send blocking mode message
async function sendBlockingMessage(message) {
    try {
        const response = await fetch(`${API_BASE}/agent/chat`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                message: message,
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
async function sendStreamingMessage(message) {
    try {
        const response = await fetch(`${API_BASE}/agent/chat`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                message: message,
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
        contentDiv.innerHTML = `<span style="color: #dc3545;">${content}</span>`;
    } else {
        contentDiv.innerHTML = marked.parse(content);
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
    messageElement.contentDiv.innerHTML = marked.parse(content) + '<span class="cursor">▋</span>';
    
    const chatMessages = document.getElementById('chatMessages');
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Finalize streaming message
function finalizeStreamingMessage(messageElement, content, error = null) {
    if (error) {
        messageElement.contentDiv.innerHTML = `<span style="color: #dc3545;">${error}</span>`;
    } else if (content !== null) {
        messageElement.contentDiv.innerHTML = marked.parse(content);
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
    const input = document.getElementById('messageInput');
    
    if (isLoading) {
        sendBtn.disabled = true;
        sendBtn.innerHTML = '<div class="loading-spinner"></div>';
        plannerBtn.disabled = true;
        input.disabled = true;
    } else {
        sendBtn.disabled = false;
        sendBtn.innerHTML = 'Send';
        plannerBtn.disabled = false;
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
    
    // Cancel ongoing stream
    if (currentStreamController) {
        currentStreamController.cancelled = true;
        currentStreamController = null;
    }
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
        } else {
            showNotification('Failed to clear memory', 'error');
        }
    } catch (error) {
        showNotification('Error: ' + error.message, 'error');
    }
}

// Refresh memory
async function refreshMemory() {
    try {
        const response = await fetch(`${API_BASE}/agent/memory`);
        if (response.ok) {
            showNotification('Memory refreshed', 'success');
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
