"""
TitanSim Mobile Server
IP: Julius Cameron Hill
Instant mobile access via Flask
"""

from flask import Flask, render_template_string, jsonify, request
from flask_cors import CORS
import json

app = Flask(__name__)
CORS(app)

# Your existing TitanSim logic imports here
# from titan_sim import YourSimulatorClass

MOBILE_TEMPLATE = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
    <title>TitanSim Mobile</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: white;
            overflow-x: hidden;
        }
        .container {
            padding: 20px;
            max-width: 600px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            padding: 30px 0;
            border-bottom: 2px solid rgba(255,255,255,0.2);
        }
        .header h1 {
            font-size: 2.5em;
            font-weight: 700;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }
        .status {
            background: rgba(255,255,255,0.1);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 20px;
            margin: 20px 0;
            border: 1px solid rgba(255,255,255,0.2);
        }
        .metric {
            display: flex;
            justify-content: space-between;
            padding: 15px 0;
            border-bottom: 1px solid rgba(255,255,255,0.1);
        }
        .metric:last-child { border-bottom: none; }
        .metric-label {
            font-weight: 600;
            opacity: 0.8;
        }
        .metric-value {
            font-size: 1.2em;
            font-weight: 700;
        }
        .controls {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
            margin: 20px 0;
        }
        button {
            background: rgba(255,255,255,0.2);
            backdrop-filter: blur(10px);
            border: 2px solid rgba(255,255,255,0.3);
            color: white;
            padding: 18px;
            border-radius: 12px;
            font-size: 1.1em;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        button:active {
            transform: scale(0.95);
            background: rgba(255,255,255,0.3);
        }
        .run-btn {
            grid-column: 1 / -1;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border: none;
            font-size: 1.3em;
            padding: 22px;
        }
        .output {
            background: rgba(0,0,0,0.3);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 20px;
            margin: 20px 0;
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
            max-height: 400px;
            overflow-y: auto;
            border: 1px solid rgba(255,255,255,0.2);
        }
        .loading {
            display: none;
            text-align: center;
            padding: 20px;
        }
        .spinner {
            border: 3px solid rgba(255,255,255,0.3);
            border-top: 3px solid white;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚡ TitanSim</h1>
            <p style="opacity: 0.8; margin-top: 10px;">Mobile Command Center</p>
        </div>

        <div class="status">
            <div class="metric">
                <span class="metric-label">Status</span>
                <span class="metric-value" id="status">Ready</span>
            </div>
            <div class="metric">
                <span class="metric-label">Simulations Run</span>
                <span class="metric-value" id="sim-count">0</span>
            </div>
            <div class="metric">
                <span class="metric-label">Last Result</span>
                <span class="metric-value" id="last-result">--</span>
            </div>
        </div>

        <div class="controls">
            <button onclick="runSimulation()">▶ Start</button>
            <button onclick="stopSimulation()">⏹ Stop</button>
            <button onclick="resetSimulation()">🔄 Reset</button>
            <button onclick="exportData()">💾 Export</button>
            <button class="run-btn" onclick="runFullCycle()">⚡ RUN FULL CYCLE</button>
        </div>

        <div class="loading" id="loading">
            <div class="spinner"></div>
            <p style="margin-top: 15px;">Running simulation...</p>
        </div>

        <div class="output" id="output">
            <div style="opacity: 0.6;">Awaiting commands...</div>
        </div>
    </div>

    <script>
        let simCount = 0;

        async function apiCall(endpoint, data = {}) {
            const response = await fetch(`/api/${endpoint}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            return await response.json();
        }

        function updateOutput(message, type = 'info') {
            const output = document.getElementById('output');
            const timestamp = new Date().toLocaleTimeString();
            const color = type === 'success' ? '#4ade80' : type === 'error' ? '#f87171' : '#ffffff';
            output.innerHTML += `<div style="color: ${color}; margin: 8px 0;">[${timestamp}] ${message}</div>`;
            output.scrollTop = output.scrollHeight;
        }

        async function runSimulation() {
            document.getElementById('status').textContent = 'Running';
            document.getElementById('loading').style.display = 'block';
            updateOutput('Starting simulation...', 'info');
            
            try {
                const result = await apiCall('run');
                simCount++;
                document.getElementById('sim-count').textContent = simCount;
                document.getElementById('last-result').textContent = result.value || 'Complete';
                document.getElementById('status').textContent = 'Complete';
                updateOutput(`Simulation complete: ${JSON.stringify(result)}`, 'success');
            } catch (e) {
                updateOutput(`Error: ${e.message}`, 'error');
                document.getElementById('status').textContent = 'Error';
            } finally {
                document.getElementById('loading').style.display = 'none';
            }
        }

        async function stopSimulation() {
            document.getElementById('status').textContent = 'Stopped';
            updateOutput('Simulation stopped', 'info');
        }

        async function resetSimulation() {
            simCount = 0;
            document.getElementById('sim-count').textContent = '0';
            document.getElementById('last-result').textContent = '--';
            document.getElementById('status').textContent = 'Ready';
            document.getElementById('output').innerHTML = '<div style="opacity: 0.6;">Reset complete. Awaiting commands...</div>';
        }

        async function exportData() {
            updateOutput('Exporting data...', 'info');
            const data = await apiCall('export');
            updateOutput('Data exported successfully', 'success');
        }

        async function runFullCycle() {
            document.getElementById('status').textContent = 'Full Cycle';
            document.getElementById('loading').style.display = 'block';
            updateOutput('⚡ INITIATING FULL CYCLE...', 'info');
            
            try {
                const result = await apiCall('full_cycle');
                simCount += result.iterations || 1;
                document.getElementById('sim-count').textContent = simCount;
                document.getElementById('last-result').textContent = 'Cycle Complete';
                document.getElementById('status').textContent = 'Ready';
                updateOutput(`✓ Full cycle complete: ${result.iterations} iterations`, 'success');
            } catch (e) {
                updateOutput(`Error: ${e.message}`, 'error');
                document.getElementById('status').textContent = 'Error';
            } finally {
                document.getElementById('loading').style.display = 'none';
            }
        }
    </script>
</body>
</html>
"""

@app.route('/')
def index():
    return render_template_string(MOBILE_TEMPLATE)

@app.route('/api/run', methods=['POST'])
def api_run():
    # Replace with your actual TitanSim logic
    # result = YourSimulatorClass().run()
    result = {"status": "success", "value": 42.5, "timestamp": "now"}
    return jsonify(result)

@app.route('/api/export', methods=['POST'])
def api_export():
    return jsonify({"status": "exported", "file": "titan_data.json"})

@app.route('/api/full_cycle', methods=['POST'])
def api_full_cycle():
    # Your full simulation cycle logic here
    return jsonify({"status": "complete", "iterations": 100})

if __name__ == '__main__':
    print("="*60)
    print("TitanSim Mobile Server")
    print("IP: Julius Cameron Hill")
    print("="*60)
    print("\n🚀 Server starting on all network interfaces...")
    print("📱 Access from your phone:")
    print("   1. Find your computer's IP address:")
    print("      Windows: ipconfig")
    print("      Mac/Linux: ifconfig")
    print("   2. On your phone, open: http://YOUR_IP:8080")
    print("   3. Add to home screen for app-like experience")
    print("="*60 + "\n")
    
    # Run on all network interfaces so phone can access
    app.run(host='0.0.0.0', port=8080, debug=True)