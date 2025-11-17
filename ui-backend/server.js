const express = require('express');
const cors = require('cors');
const bodyParser = require('body-parser');
const axios = require('axios');
const morgan = require('morgan');
require('dotenv').config();

const app = express();
const PORT = process.env.PORT || 3001;

// CORS configuration - configurable for different environments
const corsOptions = {
  origin: process.env.CORS_ORIGIN || 'http://localhost:3000',
  credentials: true,
  optionsSuccessStatus: 200
};

// Middleware
app.use(cors(corsOptions));
app.use(bodyParser.json({ limit: '10mb' }));
app.use(bodyParser.urlencoded({ extended: true }));
app.use(morgan('combined'));

// AI Agent Service configuration
const AI_AGENT_SVC_URL = process.env.AI_AGENT_SVC_URL || 'http://localhost:8080';

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'OK', timestamp: new Date().toISOString() });
});

// API endpoints to proxy to AI Agent Service

// Get agent status
app.get('/api/agent/status', async (req, res) => {
  try {
    const response = await axios.get(`${AI_AGENT_SVC_URL}/status`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching agent status:', error.message);
    res.status(500).json({ error: 'Failed to fetch agent status' });
  }
});

// Send message to agent - supports both streaming and non-streaming modes
app.post('/api/agent/chat', async (req, res) => {
  try {
    const { message, agentConfig, stream } = req.body;
    
    // Check if streaming is requested
    if (stream) {
      // Handle streaming response
      res.header('Content-Type', 'text/event-stream');
      res.header('Cache-Control', 'no-cache');
      res.header('Connection', 'keep-alive');
      res.header('Access-Control-Allow-Origin', '*');
      res.header('X-Accel-Buffering', 'no');
      
      try {
        // Forward the streaming request to AI Agent Service
        const response = await axios.post(`${AI_AGENT_SVC_URL}/chat`, {
          message,
          agentConfig,
          stream: true
        }, {
          responseType: 'stream',
          headers: {
            'Accept': 'text/event-stream'
          }
        });
        
        // Process and forward the streaming response
        response.data.on('data', (chunk) => {
          // Forward the raw SSE data to the client
          res.write(chunk);
        });
        
        response.data.on('end', () => {
          res.end();
        });
        
        response.data.on('error', (error) => {
          console.error('Streaming error:', error.message);
          if (!res.headersSent) {
            res.status(500).json({ error: 'Streaming error occurred' });
          } else {
            res.write(`event: error\ndata: ${JSON.stringify({ error: 'Streaming error occurred' })}\n\n`);
            res.end();
          }
        });
        
      } catch (error) {
        console.error('Error initiating stream:', error.message);
        if (!res.headersSent) {
          res.status(500).json({ error: 'Failed to initiate stream' });
        }
      }
    } else {
      // Handle regular non-streaming response
      const response = await axios.post(`${AI_AGENT_SVC_URL}/chat`, {
        message,
        agentConfig,
        stream: false
      });
      res.json(response.data);
    }
  } catch (error) {
    console.error('Error sending message to agent:', error.message);
    if (!res.headersSent) {
      res.status(500).json({ error: 'Failed to send message to agent' });
    }
  }
});

// Execute skill
app.post('/api/agent/skill', async (req, res) => {
  try {
    const { skillName, parameters } = req.body;
    const response = await axios.post(`${AI_AGENT_SVC_URL}/skill`, {
      skillName,
      parameters
    });
    res.json(response.data);
  } catch (error) {
    console.error('Error executing skill:', error.message);
    res.status(500).json({ error: 'Failed to execute skill' });
  }
});

// Get agent configuration
app.get('/api/agent/config', async (req, res) => {
  try {
    const response = await axios.get(`${AI_AGENT_SVC_URL}/config`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching agent config:', error.message);
    res.status(500).json({ error: 'Failed to fetch agent config' });
  }
});

// Update agent configuration
app.put('/api/agent/config', async (req, res) => {
  try {
    const config = req.body;
    const response = await axios.put(`${AI_AGENT_SVC_URL}/config`, config);
    res.json(response.data);
  } catch (error) {
    console.error('Error updating agent config:', error.message);
    res.status(500).json({ error: 'Failed to update agent config' });
  }
});

// Memory operations
app.get('/api/agent/memory', async (req, res) => {
  try {
    const response = await axios.get(`${AI_AGENT_SVC_URL}/memory`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching memory:', error.message);
    res.status(500).json({ error: 'Failed to fetch memory' });
  }
});

app.delete('/api/agent/memory', async (req, res) => {
  try {
    const response = await axios.delete(`${AI_AGENT_SVC_URL}/memory`);
    res.json(response.data);
  } catch (error) {
    console.error('Error clearing memory:', error.message);
    res.status(500).json({ error: 'Failed to clear memory' });
  }
});

// Start server
app.listen(PORT, () => {
  console.log(`UI Backend server running on port ${PORT}`);
  console.log(`Proxying requests to AI Agent Service at ${AI_AGENT_SVC_URL}`);
});
