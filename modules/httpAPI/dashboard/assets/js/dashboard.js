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


async function updateCapabilities() {
    const response = await fetch("capabilities", getFetchOptions());
    const data = await response.json();
    const versionBox = document.getElementById("version");
    const infoBox = document.getElementById("info");

    versionBox.innerText = ` ${data.Version}`;
    if (data.UserParameters.RouteChangeCounter === 0) {
        infoBox.innerText = "Displaying every BGP update received. Removing entries after 1 minute of inactivity.";
    } else {
        infoBox.innerText = `Table listing criteria: > ${data.UserParameters.RouteChangeCounter} route changes/min for ${data.UserParameters.OverThresholdTarget}min to list; ${data.UserParameters.UnderThresholdTarget}min below ${data.UserParameters.ExpiryRouteChangeCounter}/min to expire.`;
    }
    gageMaxValue = data.modHttp.gageMaxValue;

    if (data.modHttp.maxUserDefined === 0) {
        const userDefinedTrackingForm = document.querySelector("#userDefinedTracking form");
        userDefinedTrackingForm?.addEventListener("submit", (e) => {
            e.preventDefault();
            alert("This feature is disabled on this instance");
        });
    }
}

let hideZeroRateEvents = false;
document.getElementById("hideZeroRateEventsCheckbox").addEventListener("click", (e) => {
    hideZeroRateEvents = e.target.checked;
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

function getStats() {
    const sessionCountElem = document.getElementById("sessionCount");
    const noBGPFeedsElem = document.getElementById("noBGPFeeds");
    const evtSource = new EventSource("flaps/statStream");
    evtSource.addEventListener("u", (event) => {
        dataUpdate(event, true)
    });
    evtSource.addEventListener("ready", (_) => {
        liveRouteChart.update('none');
        liveFlapChart.update('none');
        loadingScreen.style.display = "none";
    });
    evtSource.addEventListener("c", (event) => {
        dataUpdate(event, false)
    });

    const avgArray = [];
    function dataUpdate(event, update) {
        try {
            const js = JSON.parse(event.data);

            const flapList = js["List"];
            const stats = js["Stats"];
            const sessionCount = js["Sessions"];
            if (sessionCount !== -1) {
                sessionCountElem.innerText = sessionCount;
            }

            if (sessionCount === 0) {
                noBGPFeedsElem.style.display = "block";
            } else {
                noBGPFeedsElem.style.display = "none";
            }

            updateList(flapList);


            addToChart(liveRouteChart, [stats["Changes"], stats["ListedChanges"]], stats["Time"], 5, update);
            addToChart(liveFlapChart, [stats["Active"]], stats["Time"], 1, update);

            avgArray.push(stats["Changes"]);
            if (avgArray.length > 50) {
                avgArray.shift();
            }

            let percentile = [...avgArray].sort((a, b) => a - b);
            percentile = percentile.slice(0, Math.ceil(percentile.length * 0.90));
            const sum = percentile.reduce((s, a) => s + a, 0);
            const avg = sum / percentile.length;
            gauge.refresh(avg / 5, gageMaxValue);

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
updateCapabilities().catch((err) => {
    console.log(err);
});

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

            data.sort((a,b) => a.RouterID.localeCompare(b.RouterID))

            data.forEach((entry) => {
                const row = document.createElement("tr");

                const cells = [
                    entry.Remote,
                    entry.RouterID,
                    entry.Hostname || "N/A",
                    toTimeElapsed(now - entry.Time)
                ];

                cells.forEach((text) => {
                    const cell = document.createElement("td");
                    cell.textContent = text;
                    cell.style.cssText = "border: 1px solid #ddd; padding: 8px; text-align: left;";
                    row.appendChild(cell);
                });

                fragment.appendChild(row);
            });

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


// Gauge-only view toggle
const gaugeOnlyToggle = document.getElementById("gaugeOnlyToggle");
if (gaugeOnlyToggle) {
    // Load saved preference
    const isGaugeOnly = localStorage.getItem("gaugeOnlyView") === "true";
    if (isGaugeOnly) {
        document.body.classList.add("gauge-only");
        gaugeOnlyToggle.textContent = "Full view";
    }

    gaugeOnlyToggle.addEventListener("click", () => {
        const currentlyGaugeOnly = document.body.classList.toggle("gauge-only");
        gaugeOnlyToggle.textContent = currentlyGaugeOnly ? "Full view" : "Gauge-only view";
        localStorage.setItem("gaugeOnlyView", currentlyGaugeOnly.toString());
    });
}