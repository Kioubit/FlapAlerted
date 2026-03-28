import "./chartjs/4.5.0/chart.umd.min.js";
import "./chartjs/chartjs-adapter-date-fns.bundle.min.js";
import {JustGage} from "./justgage/2.0.1/justgage.esm.js";

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
            borderCapStyle: "butt",
            borderDashOffset: 0.0,
            borderJoinStyle: "miter",
            pointBorderColor: "rgba(255,47,5,0.4)",
            pointBackgroundColor: "#fff",
            pointBorderWidth: 1,
            pointHoverRadius: 5,
            pointHoverBackgroundColor: "rgba(255,47,5,1)",
            pointHoverBorderColor: "rgba(220,220,220,1)",
            pointHoverBorderWidth: 2,
            pointRadius: 5,
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
            borderCapStyle: "butt",
            borderDashOffset: 0.0,
            borderJoinStyle: "miter",
            pointBorderColor: "rgba(75,192,192,1)",
            pointBackgroundColor: "#fff",
            pointBorderWidth: 1,
            pointHoverRadius: 5,
            pointHoverBackgroundColor: "rgba(75,192,192,1)",
            pointHoverBorderColor: "rgba(220,220,220,1)",
            pointHoverBorderWidth: 2,
            pointRadius: 5,
            pointHitRadius: 10,
            data: []
        },
        {
            label: "Route Changes (listed prefixes)",
            fill: false,
            backgroundColor: "rgba(15,151,3,0.4)",
            borderColor: "rgb(50,168,5)",
            borderCapStyle: "butt",
            borderDashOffset: 0.0,
            borderJoinStyle: "miter",
            pointBorderColor: "rgb(50,168,5)",
            pointBackgroundColor: "#fff",
            pointBorderWidth: 1,
            pointHoverRadius: 5,
            pointHoverBackgroundColor: "rgb(47,163,73)",
            pointHoverBorderColor: "rgba(220,220,220,1)",
            pointHoverBorderWidth: 2,
            pointRadius: 5,
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

    const errorElem = document.getElementById("peerHistoryError");
    errorElem.innerText = "";

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
    })

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
            tablePrefixes.classList.add('noDisplay');
            tablePeers.classList.remove('noDisplay');
        } else {
            activeTableView = 'prefixes';
            tablePrefixes.classList.remove('noDisplay');
            tablePeers.classList.add('noDisplay');
        }
    });
});

function updatePeers(peerList) {
    if (!peerList) {
        peersTableBody.innerHTML = '<tr><td colspan="3" class="centerText"><b>Please wait</b></td></tr>';
        return;
    }

    if (peerList.length === 0) {
        peersTableBody.innerHTML = '<tr><td colspan="3" class="centerText">No active peers detected</td></tr>';
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

        const explorerLink = (typeof explorerURLPrefixASN !== 'undefined' && explorerURLPrefixASN)
            ? `<span>&nbsp;|&nbsp;</span><a href="${explorerURLPrefixASN}${item.ASN}" target="_blank" rel="noopener noreferrer" title="Lookup ASN" class="asnExplorerLink">
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
        tr.querySelector(".asn-link").addEventListener("click", (e) => {
            e.preventDefault();
            showPeerHistory(item.ASN);
        });

        peersTableBody.appendChild(tr);
    }
}


async function fetchPeerHistory(asn) {
    const refreshBtn = document.getElementById("refreshPeerHistory");
    const errorElem = document.getElementById("peerHistoryError");

    refreshBtn.disabled = true;
    refreshBtn.classList.add("loading-btn");
    errorElem.innerText = "Fetching...";
    errorElem.style.color = "var(--text-secondary)"; // Neutral color while loading

    try {
        const response = await fetch(`peers/asn?asn=${asn}`, getFetchOptions());
        if (!response.ok) throw new Error("Failed to fetch history");

        const data = await response.json();

        if (!data || data.RateSecHistory === null || data.RateSecHistory.length === 0) {
            // Clear chart data
            peerHistoryChart.data.labels = [];
            peerHistoryChart.data.datasets[0].data = [];
            peerHistoryChart.update();

            peerHistoryChartContainer.style.display = "none";

            errorElem.innerText = "No historical data available for this ASN.";
            errorElem.style.color = "var(--accent-color)"; // Warning color
            return;
        }

        peerHistoryChartContainer.style.display = "block";
        const history = data.RateSecHistory;
        const labels = history.map((_, i) => `${history.length - 1 - i}m`);

        peerHistoryChart.data.labels = labels;
        peerHistoryChart.data.datasets[0].data = history;
        peerHistoryChart.update();

        peerHistoryDetails.innerText = `Average update rate (60min): ${data.RateSecAvg.toFixed(2)}/sec`

        errorElem.innerText = "Data updated just now.";
        errorElem.style.color = "green";

        // Fade out the success message after 3 seconds
        setTimeout(() => { if(errorElem.innerText.includes("just now")) errorElem.innerText = ""; }, 3000);

    } catch (err) {
        errorElem.innerText = `Error: ${err.message}`;
        errorElem.style.color = "#ff6b6b"; // Error color
    } finally {
        refreshBtn.disabled = false;
        refreshBtn.classList.remove("loading-btn");
    }
}


function updateList(flapList) {
    if (flapList === null) {
        prefixTable.innerHTML = '<tr><td colspan="4" class="centerText"><b>Please wait</b></td></tr>';
        return;
    }

    if (flapList.length === 0) {
        prefixTable.innerHTML = '<tr><td colspan="4" class="centerText">No flapping prefixes detected</td></tr>';
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

const loadingScreen = document.getElementById("loadingScreen");

let lastGageValue = 0;
function getStats() {
    const sessionCountElem = document.getElementById("sessionCount");
    const noBGPFeedsElem = document.getElementById("noBGPFeeds");
    const evtSource = new EventSource("flaps/statStream");
    evtSource.addEventListener("u", (event) => {
        dataUpdate(event, true)
    });
    evtSource.addEventListener("ready", (event) => {
        try {
            updateCapabilities(event.data);
        } catch (error) {
            console.error("updateCapabilities failed:", error);
        }
        loadingScreen.style.display = "none";
        liveRouteChart.update('none');
        liveFlapChart.update('none');
    });
    evtSource.addEventListener("c", (event) => {
        dataUpdate(event, false)
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
                noBGPFeedsElem.style.display = "block";
            } else {
                noBGPFeedsElem.style.display = "none";
            }

            updateList(flapList);
            updatePeers(peerList);


            addToChart(liveRouteChart, [stats["Changes"], stats["ListedChanges"]], stats["Time"], dataIntervalSec, update);
            addToChart(liveFlapChart, [stats["Active"]], stats["Time"], 1, update);

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
        loadingScreen.style.display = "none";
        handleConnectionLost(true);
        console.log(err);
    };
    evtSource.onopen = () => {
        handleConnectionLost(false);
    };
}

function toTimeElapsed(secondsIn) {
    const secondsMinute = 60;
    const secondsHour = secondsMinute * 60;
    const secondsDay = secondsHour * 24;
    const days = Math.floor(secondsIn / secondsDay);
    const hours = Math.floor((secondsIn % secondsDay) / secondsHour).toString().padStart(2, "0");
    const minutes = Math.floor((secondsIn % secondsHour) / secondsMinute).toString().padStart(2, "0");
    const seconds = Math.floor(secondsIn % secondsMinute).toString().padStart(2, "0");
    let result = "";
    if (days !== 0) {
        result += `${days}d `;
    }
    result += `${hours}:${minutes}:${seconds}`;
    return result;
}

const million = 1000000;
const billion = million * 1000;
const trillion = billion * 1000;

function truncateRouteChanges(routeChanges) {
    if (routeChanges < million) {
        return routeChanges;
    } else if (routeChanges >= million && routeChanges < billion) {
        return `${Number(routeChanges / million).toFixed(3)} million`;
    } else if (routeChanges >= billion && routeChanges < trillion) {
        return `${Number(routeChanges / billion).toFixed(3)} billion`;
    } else if (routeChanges >= trillion) {
        return `${Number(routeChanges / trillion).toFixed(2)} trillion`;
    }
    return routeChanges;
}


function handleConnectionLost(lost) {
    if (lost) {
        document.getElementById("connectionLost").style.display = "block";
    } else {
        document.getElementById("connectionLost").style.display = "none";
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
        loading.style.display = "block";
        table.style.display = "none";
        error.style.display = "none";
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
                cell.style.cssText = "border: 1px solid #ddd; padding: 8px; text-align: center;";
                row.appendChild(cell);
                fragment.appendChild(row);
            } else {
                data.sort((a,b) => a.RouterID.localeCompare(b.RouterID))

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
            loading.style.display = "none";
            table.style.display = "table";
        } catch (err) {
            loading.style.display = "none";
            error.textContent = `Error loading sessions: ${err.message}`;
            error.style.display = "block";
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

    historicalDialogOpenBtn.addEventListener('click', () => {
        historicalDialog.showModal();
        loadHistoricalEvents().then();
    });

    closeBtn.addEventListener('click', () => {
        historicalDialog.close();
    });

    async function loadHistoricalEvents() {
        // Reset view
        tableBody.innerHTML = '';
        errorElem.textContent = '';
        loadingElem.style.display = 'block';
        tableContainer.style.display = 'none';

        try {
            const response = await fetch('flaps/historical/list', getFetchOptions());

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || `Error ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();

            if (data.length === 0) {
                const row = document.createElement("tr")
                const cell = document.createElement("td")
                cell.textContent = "No historical events found";
                cell.colSpan = 3;
                cell.style.cssText = "text-align: center; padding-top: 1em;";
                row.appendChild(cell);
                tableBody.appendChild(row)
            } else {
                data.forEach(({ Prefix, Timestamp }) => {
                    const row = document.createElement("tr");
                    row.className = "historicalDialog-row";

                    const date = new Date(Timestamp * 1000).toLocaleString();
                    const url = `analyze/?prefix=${encodeURIComponent(Prefix)}&timestamp=${Timestamp}`;

                    const prefixTd = document.createElement("td");
                    prefixTd.textContent = Prefix;

                    const dateTd = document.createElement("td");
                    dateTd.textContent = date;

                    const actionTd = document.createElement("td");
                    const link = document.createElement("a");
                    link.href = url;
                    link.className = "historicalDialog-trackBtn";
                    link.textContent = "View";
                    actionTd.appendChild(link);

                    row.append(prefixTd, dateTd, actionTd);
                    tableBody.appendChild(row);
                });
            }

            loadingElem.style.display = 'none';
            tableContainer.style.display = 'block';

        } catch (error) {
            loadingElem.style.display = 'none';
            errorElem.textContent = `Failed to load events: ${error.message}`;
        }
    }
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