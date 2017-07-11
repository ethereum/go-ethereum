Chart.defaults.global.legend = {
    enabled: false
}

// Line chart for memory
var ctx = document.getElementById("memoryLineChart");
var memoryLineChart = new Chart(ctx, {
    type: 'line',
    data: {
        labels: [],
        datasets: [{
            label: "system/memory/inuse",
            backgroundColor: "rgba(38, 185, 154, 0.31)",
            borderColor: "rgba(38, 185, 154, 0.7)",
            pointBorderColor: "rgba(38, 185, 154, 0.7)",
            pointBackgroundColor: "rgba(38, 185, 154, 0.7)",
            pointHoverBackgroundColor: "#fff",
            pointHoverBorderColor: "rgba(220,220,220,1)",
            pointBorderWidth: 1,
            data: []
        }]
    },
});

// Line chart for traffic
var trafficCtx = document.getElementById("trafficLineChart");
var trafficLineChart = new Chart(trafficCtx, {
    type: 'line',
    data: {
        labels: [],
        datasets: [{
            label: "p2p/InboundTraffic",
            backgroundColor: "rgba(3, 88, 106, 0.3)",
            borderColor: "rgba(3, 88, 106, 0.70)",
            pointBorderColor: "rgba(3, 88, 106, 0.70)",
            pointBackgroundColor: "rgba(3, 88, 106, 0.70)",
            pointHoverBackgroundColor: "#fff",
            pointHoverBorderColor: "rgba(151,187,205,1)",
            pointBorderWidth: 1,
            data: []
        }]
    },
});

//TODO (kurkomisi): remove static values after debugging
const MEMORY_SAMPLE_LIMIT = 200;//{{.memorySampleLimit}}; // Maximum number of memory data samples
const TRAFFIC_SAMPLE_LIMIT = 200;//{{.trafficSampleLimit}}; // Maximum number of traffic data samples
const PROCESSOR_SAMPLE_LIMIT = 200;//{{.processorSampleLimit}}; // Maximum number of processor data samples

function updateCharts(msg) {

    // Fill the dashboard with past data
    if(msg.metrics !== undefined) {
        // Clear in case the dashboard was opened before
        memoryLineChart.data.labels = [];
        trafficLineChart.data.labels = [];
        memoryLineChart.data.datasets[0].data = [];
        trafficLineChart.data.datasets[0].data = [];

        var memory = msg.metrics.memory;
        var processor = msg.metrics.processor;
        // It is possible to get another message while filling the arrays by push(), so instead
        // put the history to the beginning of the arrays
        for (var i = memory.length - 1; i >= 0 && MEMORY_SAMPLE_LIMIT > memoryLineChart.data.labels.length; --i) {
            memoryLineChart.data.labels.unshift(memory[i].time.substring(memory[i].time.length - 5));
            trafficLineChart.data.labels.unshift(memory[i].time.substring(memory[i].time.length - 5));
            memoryLineChart.data.datasets[0].data.unshift(memory[i].value);
            trafficLineChart.data.datasets[0].data.unshift(processor[i].value);
        }
        memoryLineChart.update();
        trafficLineChart.update();
        return;
    }

    // update
    if(msg.memory !== undefined) {
        if(memoryLineChart.data.labels.length === MEMORY_SAMPLE_LIMIT) {
            memoryLineChart.data.labels.shift();
            trafficLineChart.data.labels.shift();
            memoryLineChart.data.datasets[0].data.shift();
            trafficLineChart.data.datasets[0].data.shift();
        }
        memoryLineChart.data.labels.push(msg.memory.time.substring(msg.memory.time.length - 5));
        trafficLineChart.data.labels.push(msg.memory.time.substring(msg.memory.time.length - 5));
        memoryLineChart.data.datasets[0].data.push(msg.memory.value);
        trafficLineChart.data.datasets[0].data.push(msg.processor.value);
        memoryLineChart.update();
        trafficLineChart.update();
    }
}

// Global variables to hold the current status of the dashboard
var server;

// Define a method to reconnect upon server loss
var reconnect = function() {
    server = new WebSocket("ws://" + location.host + "/api");

    server.onmessage = function(event) {
        var msg = JSON.parse(event.data);
        if (msg === null) {
            return;
        }

        updateCharts(msg)
    }

    server.onclose = function() { setTimeout(reconnect, 3000); };
}

// Establish a websocket connection to the API server
reconnect();