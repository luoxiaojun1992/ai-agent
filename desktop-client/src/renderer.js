// AI Agent Desktop - Renderer Process
// 复用现有Web UI功能并添加Electron桌面端特性

const API_BASE = 'http://localhost:3001/api';
let isLoading = false;
let currentStreamController = null;
let messageHistory = [];

// 初始化
document.addEventListener('DOMContentLoaded', async function() {
    // 加载配置
    await loadConfig();
    
    // 检查状态
    checkStatus();
    
    // 设置事件监听
    setupEventListeners();
    
    // 设置Electron菜单事件监听
    setupElectronListeners();
    
    // 聚焦输入框
    document.getElementById('messageInput').focus();
});

// 加载配置
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

// 设置事件监听
function setupEventListeners() {
    // 输入框回车发送
    document.getElementById('messageInput').addEventListener('keypress', function(e) {
        if (e.key === 'Enter' && !isLoading) {
            sendMessage();
        }
    });
    
    // 流式模式指示器
    document.querySelectorAll('input[name="chatMode"]').forEach(radio => {
        radio.addEventListener('change', function() {
            updateStreamIndicator(this.value === 'streaming');
        });
    });
}

// 设置Electron菜单事件监听
function setupElectronListeners() {
    if (!window.electronAPI) return;
    
    // 新建会话
    window.electronAPI.onMenuNewChat((event) => {
        clearChat();
    });
    
    // 清除聊天
    window.electronAPI.onMenuClearChat((event) => {
        clearChat();
    });
    
    // API设置
    window.electronAPI.onMenuApiSettings((event) => {
        showConfig();
    });
    
    // 刷新配置
    window.electronAPI.onMenuRefreshConfig((event) => {
        refreshConfig();
    });
}

// 更新流式模式指示器
function updateStreamIndicator(isStreaming) {
    const indicator = document.querySelector('.streaming-indicator');
    if (indicator) {
        indicator.style.display = isStreaming ? 'inline-block' : 'none';
    }
}

// 检查API状态
async function checkStatus() {
    try {
        const response = await fetch(`${API_BASE}/agent/status`);
        if (response.ok) {
            const data = await response.json();
            updateStatus(true, '已连接');
            refreshConfig();
        } else {
            updateStatus(false, '连接失败');
        }
    } catch (error) {
        updateStatus(false, '未连接');
        console.error('Status check error:', error);
    }
}

// 更新状态显示
function updateStatus(connected, text) {
    const dot = document.getElementById('statusDot');
    const statusText = document.getElementById('statusText');
    const apiStatus = document.getElementById('api-status');
    
    dot.className = 'status-dot' + (connected ? ' connected' : ' error');
    statusText.textContent = text;
    apiStatus.textContent = 'API: ' + text;
}

// 发送消息
async function sendMessage() {
    const input = document.getElementById('messageInput');
    const message = input.value.trim();
    const mode = document.querySelector('input[name="chatMode"]:checked').value;
    
    if (!message || isLoading) return;
    
    isLoading = true;
    updateUIState();
    
    // 添加用户消息
    addMessage(message, 'user');
    input.value = '';
    
    // 保存到历史记录
    messageHistory.push({ role: 'user', content: message });
    
    if (mode === 'streaming') {
        await sendStreamingMessage(message);
    } else {
        await sendBlockingMessage(message);
    }
    
    isLoading = false;
    updateUIState();
}

// 发送阻塞模式消息
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
            addMessage('错误：无法获取 AI 响应', 'agent', true);
        }
    } catch (error) {
        addMessage('错误：' + error.message, 'agent', true);
        updateStatus(false, '连接错误');
    }
}

// 发送流式消息
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
            addMessage('错误：无法启动流式响应', 'agent', true);
            return;
        }
        
        // 创建流式消息元素
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
                                finalizeStreamingMessage(messageElement, null, '错误：' + (data.error || '未知错误'));
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
            finalizeStreamingMessage(messageElement, null, '流式错误：' + error.message);
        } finally {
            currentStreamController = null;
        }
        
    } catch (error) {
        addMessage('错误：' + error.message, 'agent', true);
        updateStatus(false, '连接错误');
    }
}

// 添加消息到聊天区域
function addMessage(content, sender, isError = false) {
    const chatMessages = document.getElementById('chatMessages');
    
    // 移除欢迎消息
    const welcomeMsg = chatMessages.querySelector('.welcome-message');
    if (welcomeMsg) {
        welcomeMsg.remove();
    }
    
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${sender}`;
    
    const headerDiv = document.createElement('div');
    headerDiv.className = 'message-header';
    headerDiv.innerHTML = sender === 'user' 
        ? '<span>我</span>'
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
    
    // 滚动到底部
    chatMessages.scrollTop = chatMessages.scrollHeight;
    
    return messageDiv;
}

// 添加流式消息
function addStreamingMessage() {
    const chatMessages = document.getElementById('chatMessages');
    
    // 移除欢迎消息
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

// 更新流式消息
function updateStreamingMessage(messageElement, content) {
    messageElement.contentDiv.innerHTML = marked.parse(content) + '<span class="cursor">▋</span>';
    
    const chatMessages = document.getElementById('chatMessages');
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// 完成流式消息
function finalizeStreamingMessage(messageElement, content, error = null) {
    if (error) {
        messageElement.contentDiv.innerHTML = `<span style="color: #dc3545;">${error}</span>`;
    } else if (content !== null) {
        messageElement.contentDiv.innerHTML = marked.parse(content);
    }
    
    // 移除流式指示器
    const indicator = messageElement.messageDiv.querySelector('.streaming-indicator');
    if (indicator) {
        indicator.remove();
    }
}

// 更新UI状态
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
        sendBtn.innerHTML = '发送';
        plannerBtn.disabled = false;
        input.disabled = false;
        input.focus();
    }
}

// 清除聊天
function clearChat() {
    const chatMessages = document.getElementById('chatMessages');
    chatMessages.innerHTML = `
        <div class="welcome-message">
            <h2>👋 欢迎使用 AI Agent Desktop</h2>
            <p>这是一个功能强大的 AI Agent 桌面客户端。您可以与 AI 进行对话，使用各种技能，或者让 Agent Planner 帮您规划和执行任务。</p>
        </div>
    `;
    
    messageHistory = [];
    
    // 取消正在进行的流
    if (currentStreamController) {
        currentStreamController.cancelled = true;
        currentStreamController = null;
    }
}

// 运行 Agent Planner
async function runPlanner() {
    const input = document.getElementById('messageInput');
    const task = input.value.trim();
    
    if (!task) {
        input.focus();
        return;
    }
    
    const mode = document.querySelector('input[name="chatMode"]:checked').value;
    const plannerMessage = "请为以下任务创建详细计划，然后逐步执行。尽可能使用 MCP 技能。任务：" + task;
    
    isLoading = true;
    updateUIState();
    
    // 添加用户消息
    addMessage("执行计划任务：" + task, 'user');
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

// 刷新配置
async function refreshConfig() {
    try {
        const response = await fetch(`${API_BASE}/agent/config`);
        if (response.ok) {
            const data = await response.json();
            const configDisplay = document.getElementById('configDisplay');
            configDisplay.innerHTML = `
                <div class="config-item">
                    <label>聊天模型</label>
                    <input type="text" value="${data.chatModel || 'N/A'}" readonly>
                </div>
                <div class="config-item">
                    <label>嵌入模型</label>
                    <input type="text" value="${data.embeddingModel || 'N/A'}" readonly>
                </div>
                <div class="config-item">
                    <label>模式</label>
                    <input type="text" value="${data.agentMode || 'N/A'}" readonly>
                </div>
                <div class="config-item">
                    <label>角色</label>
                    <input type="text" value="${data.character || 'N/A'}" readonly>
                </div>
            `;
        }
    } catch (error) {
        document.getElementById('configDisplay').innerHTML = 
            '<p style="color: #dc3545;">加载配置失败</p>';
    }
}

// 清除内存
async function clearMemory() {
    try {
        const response = await fetch(`${API_BASE}/agent/memory`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            showNotification('内存已清除', 'success');
        } else {
            showNotification('清除内存失败', 'error');
        }
    } catch (error) {
        showNotification('错误：' + error.message, 'error');
    }
}

// 刷新内存
async function refreshMemory() {
    try {
        const response = await fetch(`${API_BASE}/agent/memory`);
        if (response.ok) {
            showNotification('内存已刷新', 'success');
        } else {
            showNotification('刷新内存失败', 'error');
        }
    } catch (error) {
        showNotification('错误：' + error.message, 'error');
    }
}

// 显示配置面板
function showConfig() {
    document.getElementById('configPanel').classList.add('active');
    refreshConfig();
}

// 显示聊天区域
function showChat() {
    document.getElementById('configPanel').classList.remove('active');
}

// 保存API配置
async function saveApiConfig() {
    const apiBase = document.getElementById('apiBaseInput').value.trim();
    
    if (window.electronAPI) {
        const config = await window.electronAPI.getConfig();
        config.apiBase = apiBase;
        await window.electronAPI.saveConfig(config);
    }
    
    showNotification('配置已保存', 'success');
    showChat();
    checkStatus();
}

// 导出聊天记录
async function exportChat() {
    if (messageHistory.length === 0) {
        showNotification('没有可导出的聊天记录', 'error');
        return;
    }
    
    const content = messageHistory.map(m => {
        const role = m.role === 'user' ? '用户' : 'AI Agent';
        return `[${role}]\n${m.content}\n`;
    }).join('\n---\n\n');
    
    if (window.electronAPI) {
        const result = await window.electronAPI.showSaveDialog({
            defaultPath: `chat-export-${new Date().toISOString().slice(0, 10)}.txt`,
            filters: [
                { name: '文本文件', extensions: ['txt'] },
                { name: '所有文件', extensions: ['*'] }
            ]
        });
        
        if (!result.canceled && result.filePath) {
            const writeResult = await window.electronAPI.writeFile(result.filePath, content);
            if (writeResult.success) {
                showNotification('聊天记录已导出', 'success');
            } else {
                showNotification('导出失败：' + writeResult.error, 'error');
            }
        }
    } else {
        // 浏览器环境：使用下载
        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `chat-export-${new Date().toISOString().slice(0, 10)}.txt`;
        a.click();
        URL.revokeObjectURL(url);
    }
}

// 显示通知
function showNotification(message, type = 'info') {
    // 创建通知元素
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
    
    // 3秒后移除
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// 添加动画样式
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
