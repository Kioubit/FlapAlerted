import {
    Chart, LineController, LineElement, PointElement, LinearScale, TimeScale, Tooltip, Legend, Filler, CategoryScale
} from "chart.js";

import 'chartjs-adapter-date-fns';
import JustGage from "justgage";
import {getRelativeTime, toTimeElapsed, truncateRouteChanges} from "./util";

Chart.register(
    CategoryScale,
    LineController,
    LineElement,
    PointElement,
    LinearScale,
    TimeScale,
    Tooltip,
    Legend,
    Filler
);

function getFetchOptions() {
    return {
        headers: {
            'X-AS': Math.floor(Date.now() / 1000).toString()
        }
    };
}

let gageMaxValue = 400;
let gageMaxValueModifier = 1;
let gageDisableDynamic = false;
const gauge = new JustGage({
    id: "justgage",
    value: 0,
    min: 0,
    max: gageMaxValue,
    label: "Average Route Changes",
    decimals: 2,
    gaugeWidthScale: 0.2,
    pointer: true,
    relativeGaugeSize: true,
    customSectors: {
        // lo and hi values are in %
        percents: true,
        ranges: [
            {
                color: "#43bf58",
                lo: 0,
                hi: 20
            },
            {
                color: "#f7bc08",
                lo: 21,
                hi: 70
            },
            {
                color: "#ff3b30",
                lo: 71,
                hi: 100
            }
        ]
    }
});

const dataFlapCount = {
    labels: [],
    datasets: [
        {
            label: "Count of active prefixes",
            fill: false,
            backgroundColor: "rgba(255,47,5,0.4)",
            borderColor: "rgba(255,47,5,1)",
            pointBorderWidth: 1,
            pointBackgroundColor: "#fff",
            pointRadius: 4,
            pointHitRadius: 10,
            data: []
        }
    ]
};

const dataRouteChange = {
    labels: [],
    datasets: [
        {
            label: "Route Changes",
            fill: false,
            backgroundColor: "rgba(75,192,192,0.4)",
            borderColor: "rgba(75,192,192,1)",
            pointBorderWidth: 1,
            pointBackgroundColor: "#fff",
            pointRadius: 4,
            pointHitRadius: 10,
            data: []
        },
        {
            label: "Route Changes (listed prefixes)",
            fill: false,
            backgroundColor: "rgba(15,151,3,0.4)",
            borderColor: "rgb(50,168,5)",
            pointBorderWidth: 1,
            pointBackgroundColor: "#fff",
            pointRadius: 4,
            pointHitRadius: 10,
            data: []
        }
    ]
};

const ctxFlapCount = document.getElementById("chartFlapCount").getContext("2d");
const ctxRoute = document.getElementById("chartRoute").getContext("2d");

const liveFlapChart = new Chart(
    ctxFlapCount,
    {
        type: "line",
        data: dataFlapCount,
        options: {
            scales: {
                x: {
                    type: "time",
                    time: {
                        unit: "minute",
                        displayFormats: {
                            minute: "HH:mm"
                        },
                        tooltipFormat: "HH:mm:ss"
                    }
                },
                y: {
                    suggestedMin: 0,
                    suggestedMax: 10
                }
            },
            maintainAspectRatio: false
        }
    }
);


const liveRouteChart = new Chart(
    ctxRoute,
    {
        type: "line",
        data: dataRouteChange,
        options: {
            scales: {
                x: {
                    type: "time",
                    time: {
                        unit: "minute",
                        displayFormats: {
                            minute: "HH:mm"
                        },
                        tooltipFormat: "HH:mm:ss"
                    }
                },
                y: {
                    suggestedMin: 0,
                    suggestedMax: 15
                }
            },
            maintainAspectRatio: false,
            plugins: {
                tooltip: {
                    callbacks: {
                        label: (context) => `${context.dataset.label}: ${context.parsed.y}/sec`
                    }
                }
            }
        }
    }
);

const ctxImport = document.getElementById("chartImportCount").getContext("2d");

const dataImportCount = {
    labels: [],
    datasets: [
        {
            label: "Total Imported Routes",
            fill: true,
            backgroundColor: "rgb(37 99 235 / 10%)",
            borderColor: "rgb(37 99 235)",
            pointBorderWidth: 1,
            pointBackgroundColor: "#fff",
            pointRadius: 4,
            pointHitRadius: 10,
            data: []
        }
    ]
};


const liveImportChart = new Chart(
    ctxImport,
    {
        type: "line",
        data: dataImportCount,
        options: {
            scales: {
                x: {
                    type: "time",
                    time: {
                        unit: "minute",
                        displayFormats: { minute: "HH:mm" },
                        tooltipFormat: "HH:mm:ss"
                    }
                },
                y: {
                    beginAtZero: false,
                }
            },
            maintainAspectRatio: false
        }
    }
);

let peerHistoryChart = null;
const peerHistoryChartContainer = document.getElementById("chartPeerHistoryContainer");
const ctxPeerHistory = document.getElementById("chartPeerHistory").getContext("2d");

function initPeerChart() {
    if (peerHistoryChart) return;
    peerHistoryChart = new Chart(ctxPeerHistory, {
        type: "line",
        data: {
            labels: [],
            datasets: [{
                label: "Changes/sec",
                data: [],
                borderColor: "rgba(37, 99, 235, 1)",
                backgroundColor: "rgba(37, 99, 235, 0.1)",
                fill: true,
                tension: 0.3
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: { display: true, title: { display: true, text: 'Minutes Ago' } },
                y: { beginAtZero: true }
            }
        }
    });
}

const peerHistoryDialog = document.getElementById("peerHistoryDialog");
const peerHistoryASNLabel = document.getElementById("peerHistoryASN");
const peerHistoryDetails = document.getElementById("peerHistoryDetails");

// Variable to store the currently viewed ASN so the refresh button knows what to fetch
let currentHistoryASN = null;

async function showPeerHistory(asn) {
    currentHistoryASN = asn;
    peerHistoryASNLabel.innerText = asn;

    peerHistoryDetails.innerText = "";

    const statusElem = document.getElementById("peerHistoryStatus");
    statusElem.innerText = "";

    peerHistoryDialog.showModal();
    initPeerChart();

    // Explicitly reset chart data so old data doesn't flicker
    peerHistoryChart.data.labels = [];
    peerHistoryChart.data.datasets[0].data = [];
    peerHistoryChart.update();

    await fetchPeerHistory(asn);
}

// Add the Refresh Button event listener
document.getElementById("refreshPeerHistory").addEventListener("click", () => {
    if (currentHistoryASN) {
        fetchPeerHistory(currentHistoryASN);
    }
});
document.getElementById("closePeerHistoryDialog").onclick = () => peerHistoryDialog.close();

let explorerURLPrefixASN = null;
function updateCapabilities(response) {
    const data = JSON.parse(response);
    const versionBox = document.getElementById("version");
    const infoBox = document.getElementById("info");

    versionBox.innerText = ` ${data.Version}`;
    if (data.UserParameters.RouteChangeCounter === 0) {
        infoBox.innerText = "Displaying every BGP update received. Removing entries after 1 minute of inactivity.";
    } else {
        infoBox.innerText = `Table listing criteria: > ${data.UserParameters.RouteChangeCounter} route changes/min for ${data.UserParameters.OverThresholdTarget}min to list; ${data.UserParameters.UnderThresholdTarget}min below ${data.UserParameters.ExpiryRouteChangeCounter}/min to expire.`;
    }

    historicalDialogOpenBtn.disabled = data.HistoryProviderAvailable === false;
    explorerURLPrefixASN = data.modHttp.explorerUrlPrefixASN;

    gageDisableDynamic = data.modHttp.gageDisableDynamic;
    gageMaxValue = data.modHttp.gageMaxValue;
    gauge.refresh(lastGageValue, gageMaxValue * gageMaxValueModifier);

    if (data.modHttp.maxUserDefined === 0) {
        const userDefinedTrackingForm = document.querySelector("#userDefinedTracking form");
        const submit = userDefinedTrackingForm.querySelector("button[type='submit']");
        submit.disabled = true;
        userDefinedTrackingForm?.addEventListener("submit", (e) => {
            e.preventDefault();
            alert("This feature is disabled on this instance");
        });
    }
}

const lookupPeerForm = document.querySelector("#peerLookup form");
lookupPeerForm?.addEventListener("submit", (e) => {
    e.preventDefault();
    const value = new FormData(lookupPeerForm).get("asn");
    showPeerHistory(value).then();
});

let hideZeroRateEvents = false;
if (localStorage.getItem("fa_hideZeroRate") === "true") {
    hideZeroRateEvents = true;
    document.getElementById("hideZeroRateEventsCheckbox").checked = true;
}
document.getElementById("hideZeroRateEventsCheckbox").addEventListener("click", (e) => {
    hideZeroRateEvents = e.target.checked;
    localStorage.setItem("fa_hideZeroRate", hideZeroRateEvents.toString());
});


function addToChart(liveChart, points, unixTime, dataInterval, update) {
    const timestamp = unixTime * 1000;
    const shouldShift = liveChart.data.labels.length > 50;

    liveChart.data.datasets.forEach((dataset, i) => {
        if (i >= points.length) {
            return;
        }
        dataset.data.push(points[i] / dataInterval);
        if (shouldShift) {
            dataset.data.shift();
        }
    });

    liveChart.data.labels.push(timestamp);
    if (shouldShift) {
        liveChart.data.labels.shift();
    }
    if (update) {
        liveChart.update();
    }
}

const prefixTable = document.getElementById("prefixTableBody");
const peersTableBody = document.getElementById("peersTableBody");
const tablePrefixes = document.getElementById("tablePrefixes");
const tablePeers = document.getElementById("tablePeers");

let activeTableView = 'prefixes';
document.querySelectorAll(".toggle-view-btn").forEach(btn => {
    btn.addEventListener("click", () => {
        if (activeTableView === 'prefixes') {
            activeTableView = 'peers';
            tablePrefixes.classList.add('d-none');
            tablePeers.classList.remove('d-none');
        } else {
            activeTableView = 'prefixes';
            tablePrefixes.classList.remove('d-none');
            tablePeers.classList.add('d-none');
        }
    });
});

function updatePeers(peerList) {
    if (!peerList) {
        peersTableBody.innerHTML = '<tr><td colspan="3" class="text-center"><b>Please wait</b></td></tr>';
        return;
    }

    if (peerList.length === 0) {
        peersTableBody.innerHTML = '<tr><td colspan="3" class="text-center">No active peers detected</td></tr>';
        return;
    }

    // Sort array by highest RateSec
    const sorted = [...peerList].sort((a, b) => b.RateSecAvg - a.RateSecAvg);
    const limit = Math.min(101, sorted.length);

    peersTableBody.innerHTML = "";
    for (let i = 0; i < limit; i++) {
        const item = sorted[i];
        if (hideZeroRateEvents && item.RateSec < 1) continue;

        const rowClass = item.RateSec < 1 ? ' class="inactive"' : "";
        const rateDisplayAvg = item.RateSecAvg !== -1 ? `${item.RateSecAvg.toFixed(2)}/s` : "..";
        const rateDisplay = item.RateSec !== -1 ? `${item.RateSec}/s` : "..";

        const tr = document.createElement("tr");
        if (rowClass) tr.className = "inactive";

        const explorerLink = (typeof explorerURLPrefixASN !== 'undefined' && explorerURLPrefixASN) ?
            `<span>&nbsp;|&nbsp;</span><a href="${explorerURLPrefixASN}${item.ASN}" target="_blank" rel="noopener noreferrer" title="Lookup ASN" class="asnExplorerLink">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                    <polyline points="15 3 21 3 21 9"></polyline>
                    <line x1="10" y1="14" x2="21" y2="3"></line>
                </svg>
                </a>
            `
            : "";

        tr.innerHTML = `
        <td><a href="#" class="asn-link" data-asn="${item.ASN}">${item.ASN}</a>${explorerLink}</td>
        <td>${rateDisplayAvg}</td>
        <td>${rateDisplay}</td>
    `;
        const asnLink = tr.querySelector(".asn-link");
        asnLink.addEventListener("click", (e) => {
            e.preventDefault();
            showPeerHistory(item.ASN).then();
        });

        asnLink.addEventListener("auxclick", (e) => e.preventDefault());

        peersTableBody.appendChild(tr);
    }
}


async function fetchPeerHistory(asn) {
    const refreshBtn = document.getElementById("refreshPeerHistory");
    const statusElem = document.getElementById("peerHistoryStatus");

    refreshBtn.disabled = true;
    refreshBtn.classList.add("loading-btn");
    statusElem.removeAttribute("data-result-type");
    statusElem.innerText = "Fetching...";

    try {
        const response = await fetch(`peers/asn?asn=${asn}`, getFetchOptions());
        if (!response.ok) throw new Error("Failed to fetch history");

        const data = await response.json();

        if (!data || data.RateSecHistory === null || data.RateSecHistory.length === 0) {
            // Clear chart data
            peerHistoryChart.data.labels = [];
            peerHistoryChart.data.datasets[0].data = [];
            peerHistoryChart.update('none');

            peerHistoryChartContainer.classList.add('d-none');

            statusElem.innerText = "No historical data available for this ASN";
            return;
        }

        peerHistoryChartContainer.classList.remove('d-none');
        const history = data.RateSecHistory;
        peerHistoryChart.data.labels = history.map((_, i) => `${history.length - 1 - i}m`);
        peerHistoryChart.data.datasets[0].data = history;
        peerHistoryChart.update('none');

        peerHistoryDetails.innerText = `Average update rate (60min): ${data.RateSecAvg.toFixed(2)}/sec`;

        statusElem.innerText = "Data updated just now.";
        statusElem.setAttribute("data-result-type", "success");

        // Fade out the success message after 3 seconds
        setTimeout(() => { if(statusElem.innerText.includes("just now")) statusElem.innerText = ""; }, 3000);

    } catch (err) {
        statusElem.innerText = `Error: ${err.message}`;
        statusElem.setAttribute("data-result-type", "error")
    } finally {
        refreshBtn.disabled = false;
        refreshBtn.classList.remove("loading-btn");
    }
}


function updateList(flapList) {
    if (flapList === null) {
        prefixTable.innerHTML = '<tr><td colspan="4" class="text-center"><b>Please wait</b></td></tr>';
        return;
    }

    if (flapList.length === 0) {
        prefixTable.innerHTML = '<tr><td colspan="4" class="text-center">No flapping prefixes detected</td></tr>';
        return;
    }

    const unixTime = Math.floor(Date.now() / 1000);
    const rows = [];

    flapList.sort((a, b) => b.TotalCount - a.TotalCount);
    const limit = Math.min(101, flapList.length);

    for (let i = 0; i < limit; i++) {
        const item = flapList[i];

        if (hideZeroRateEvents && item.RateSec < 1) continue;

        const rowClass = item.RateSec < 1 ? ' class="inactive"' : "";
        const rateDisplay = item.RateSec !== -1 ? `${item.RateSec}/s` : "..";

        rows.push(`<tr${rowClass}>
            <td><a target="_blank" href='analyze/?prefix=${encodeURIComponent(item.Prefix)}'>${item.Prefix}</a></td>
            <td>${toTimeElapsed(unixTime - item.FirstSeen)}</td>
            <td>${truncateRouteChanges(item.TotalCount)}</td>
            <td>${rateDisplay}</td>
        </tr>`);
    }

    prefixTable.innerHTML = rows.join('');
}

const loadingScreen = document.getElementById("loading-screen");

let lastGageValue = 0;
function getStats() {
    const sessionCountElem = document.getElementById("sessionCount");
    const noBGPFeedsElem = document.getElementById("noBGPFeeds");
    const evtSource = new EventSource("flaps/statStream");
    evtSource.addEventListener("u", (event) => {
        dataUpdate(event, true);
    });
    evtSource.addEventListener("ready", (event) => {
        try {
            updateCapabilities(event.data);
        } catch (error) {
            console.error("updateCapabilities failed:", error);
        }
        loadingScreen.classList.add("d-none");
        liveRouteChart.update('none');
        liveFlapChart.update('none');
        liveImportChart.update('none');
    });
    evtSource.addEventListener("c", (event) => {
        dataUpdate(event, false);
    });

    const dataIntervalSec = 5;
    const avgArray = [];
    function dataUpdate(event, update) {
        try {
            const js = JSON.parse(event.data);

            const flapList = js["List"];
            const peerList = js["ListPeers"];
            const stats = js["Stats"];
            const sessionCount = js["Sessions"];
            if (sessionCount !== -1) {
                sessionCountElem.innerText = sessionCount;
                if (gageDisableDynamic) {
                    gageMaxValueModifier = 1;
                } else if (sessionCount > 0) {
                    gageMaxValueModifier = sessionCount;
                }
            }

            if (sessionCount === 0) {
                noBGPFeedsElem.classList.remove("d-none");
            } else {
                noBGPFeedsElem.classList.add("d-none");
            }

            updateList(flapList);
            updatePeers(peerList);


            addToChart(liveRouteChart, [stats["Changes"], stats["ListedChanges"]], stats["Time"], dataIntervalSec, update);
            addToChart(liveFlapChart, [stats["Active"]], stats["Time"], 1, update);
            addToChart(liveImportChart, [stats["RouteCount"]], stats["Time"], 1, update);

            avgArray.push(stats["Changes"]);
            if (avgArray.length > 50) {
                avgArray.shift();
            }

            let percentile = [...avgArray].sort((a, b) => a - b);
            percentile = percentile.slice(0, Math.ceil(percentile.length * 0.90));
            const sum = percentile.reduce((s, a) => s + a, 0);
            const avg = sum / percentile.length;
            lastGageValue = avg / dataIntervalSec;
            gauge.refresh(avg / dataIntervalSec, gageMaxValue * gageMaxValueModifier);

        } catch (err) {
            console.log(err);
        }
    }

    evtSource.onerror = (err) => {
        loadingScreen.classList.add("d-none");
        handleConnectionLost(true);
        console.log(err);
    };
    evtSource.onopen = () => {
        handleConnectionLost(false);
    };
}

function handleConnectionLost(lost) {
    const lostErrorElem = document.getElementById("connectionLost");
    if (lost) {
        lostErrorElem.classList.remove("d-none");
    } else {
        lostErrorElem.classList.add("d-none");
    }
}

getStats();

{
    const dialog = document.getElementById("sessionsDialog");
    const loading = document.getElementById("sessionsLoading");
    const table = document.getElementById("sessionsTable");
    const tbody = document.getElementById("sessionsTableBody");
    const error = document.getElementById("sessionsError");

    document.getElementById("closeSessionsDialog")?.addEventListener("click", () => {
        dialog.close();
    });

    document.getElementById("sessionCountLink").onclick = async (ev) => {
        ev.preventDefault();
        loading.classList.remove("d-none");
        table.classList.add("d-none");
        error.classList.add("d-none");
        tbody.innerHTML = "";

        dialog.showModal();

        try {
            const response = await fetch("sessions", getFetchOptions());
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            const data = await response.json();

            const now = Math.floor(Date.now() / 1000);
            const fragment = document.createDocumentFragment();

            if (data.length === 0) {
                const row = document.createElement("tr");
                const cell = document.createElement("td");
                cell.textContent = "No established sessions";
                cell.colSpan = 5;
                row.appendChild(cell);
                fragment.appendChild(row);
            } else {
                data.sort((a,b) => a.RouterID.localeCompare(b.RouterID));

                data.forEach((entry) => {
                    const row = document.createElement("tr");

                    const cells = [
                        entry.Remote,
                        entry.RouterID,
                        entry.Hostname || "--",
                        toTimeElapsed(now - entry.EstablishTime),
                        Number(entry.ImportCount).toLocaleString()
                    ];

                    cells.forEach((text) => {
                        const cell = document.createElement("td");
                        cell.textContent = text;
                        cell.classList.add("SessionsDialog-bodyCell");
                        row.appendChild(cell);
                    });

                    fragment.appendChild(row);
                });
                const total = data.reduce((sum, entry) => sum + Number(entry.ImportCount || 0), 0);
                document.getElementById("totalImportCount").textContent = total.toLocaleString();
            }

            tbody.appendChild(fragment);
            loading.classList.add("d-none");
            table.classList.remove("d-none");
        } catch (err) {
            loading.classList.add("d-none");
            error.textContent = `Error loading sessions: ${err.message}`;
            error.classList.remove("d-none");
        }
    };
}

const historicalDialogOpenBtn = document.getElementById('viewHistoricalEvents');

{
    const historicalDialog = document.getElementById('historicalDialog');
    const closeBtn = document.getElementById('closeHistoricalDialog');
    const tableBody = document.getElementById('historicalTableBody');
    const loadingElem = document.getElementById('historicalLoading');
    const errorElem = document.getElementById('historicalError');
    const tableContainer = document.getElementById('historicalTableContainer');
    const toggleFullTimestamp = document.getElementById('historicalToggleFullTimestamp');

    let showFullTimestamps = false;
    let loading = false;

    historicalDialogOpenBtn.addEventListener('click', async () => {
        historicalDialog.showModal();

        if (!loading) {
            await loadHistoricalEvents();
        }
    });

    closeBtn.addEventListener('click', () => {
        historicalDialog.close();
    });

    function formatTimestamp(timestamp) {
        const date = new Date(timestamp * 1000);

        return showFullTimestamps
            ? date.toLocaleString()
            : getRelativeTime(date);
    }

    function updateVisibleTimestamps() {
        for (const cell of tableBody.querySelectorAll('[data-timestamp]')) {
            cell.textContent = formatTimestamp(Number(cell.dataset.timestamp));
        }
    }

    async function loadHistoricalEvents() {
        loading = true;

        tableBody.replaceChildren();
        errorElem.textContent = '';
        loadingElem.classList.remove('d-none');
        tableContainer.classList.add('d-none');

        try {
            const response = await fetch(
                'flaps/historical/list',
                getFetchOptions()
            );

            if (!response.ok) {
                throw new Error(
                    (await response.text()) ||
                    `Error ${response.status}: ${response.statusText}`
                );
            }

            const data = await response.json();

            if (data.length === 0) {
                const row = document.createElement('tr');
                const cell = document.createElement('td');

                cell.textContent = 'No historical events found';
                cell.colSpan = 4;

                row.appendChild(cell);
                tableBody.appendChild(row);

                tableContainer.classList.remove("d-none");
                return;
            }

            const fragment = document.createDocumentFragment();

            for (const { Prefix, Timestamp, AvgChangeRate } of data) {
                const row = document.createElement('tr');
                row.className = 'historicalDialog-row';

                const prefixTd = document.createElement('td');
                prefixTd.textContent = Prefix;

                const dateTd = document.createElement('td');
                dateTd.dataset.timestamp = String(Timestamp);
                dateTd.textContent = formatTimestamp(Timestamp);

                const avgChangeRateTd = document.createElement('td');
                avgChangeRateTd.textContent = AvgChangeRate.toFixed(2);

                const actionTd = document.createElement('td');

                const link = document.createElement('a');
                link.href =
                    `analyze/?prefix=${encodeURIComponent(Prefix)}` +
                    `&timestamp=${Timestamp}`;
                link.textContent = 'View';

                actionTd.appendChild(link);

                row.append(
                    prefixTd,
                    dateTd,
                    avgChangeRateTd,
                    actionTd
                );

                fragment.appendChild(row);
            }

            tableBody.appendChild(fragment);
            tableContainer.classList.remove('d-none');
        } catch (error) {
            errorElem.textContent =
                `Failed to load events: ${error.message}`;
        } finally {
            loading = false;
            loadingElem.classList.add('d-none');
        }
    }

    toggleFullTimestamp.addEventListener('change', () => {
        showFullTimestamps = toggleFullTimestamp.checked;
        updateVisibleTimestamps();
    });
}

// Gauge-only view toggle
const gaugeOnlyToggle = document.getElementById("gaugeOnlyToggle");
if (gaugeOnlyToggle) {
    // Load saved preference
    const isGaugeOnly = localStorage.getItem("fa_gaugeOnlyView") === "true";
    if (isGaugeOnly) {
        document.body.classList.add("gauge-only");
        gaugeOnlyToggle.textContent = "Full view";
    }

    gaugeOnlyToggle.addEventListener("click", () => {
        const currentlyGaugeOnly = document.body.classList.toggle("gauge-only");
        gaugeOnlyToggle.textContent = currentlyGaugeOnly ? "Full view" : "Gauge-only view";
        localStorage.setItem("fa_gaugeOnlyView", currentlyGaugeOnly.toString());
    });
}