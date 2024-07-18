const gauge = new JustGage({
    id: "justgage",
    value: 0,
    min: 0,
    max: 200,
    label: "Average Route Changes",
    decimals: 2,
    gaugeWidthScale: 0.2,
    pointer: true,
    relativeGaugeSize: true,
    customSectors: {
        percents: true, // lo and hi values are in %
        ranges: [{
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
            }]
    }
});

const dataFlapCount = {
    labels: [],
    datasets: [
        {
            label: "Count of active prefixes",
            fill: false,
            lineTension: 0.1,
            backgroundColor: "rgba(255,47,5,0.4)",
            borderColor: "rgba(255,47,5,1)",
            borderCapStyle: 'butt',
            borderDash: [],
            borderDashOffset: 0.0,
            borderJoinStyle: 'miter',
            pointBorderColor: "rgba(75,192,192,1)",
            pointBackgroundColor: "#fff",
            pointBorderWidth: 1,
            pointHoverRadius: 5,
            pointHoverBackgroundColor: "rgba(75,192,192,1)",
            pointHoverBorderColor: "rgba(220,220,220,1)",
            pointHoverBorderWidth: 2,
            pointRadius: 5,
            pointHitRadius: 10,
            data: [],
        }
    ]
};

const dataRouteChange = {
    labels: [],
    datasets: [
        {
            label: "Route Changes",
            fill: false,
            lineTension: 0.1,
            backgroundColor: "rgba(75,192,192,0.4)",
            borderColor: "rgba(75,192,192,1)",
            borderCapStyle: 'butt',
            borderDash: [],
            borderDashOffset: 0.0,
            borderJoinStyle: 'miter',
            pointBorderColor: "rgba(75,192,192,1)",
            pointBackgroundColor: "#fff",
            pointBorderWidth: 1,
            pointHoverRadius: 5,
            pointHoverBackgroundColor: "rgba(75,192,192,1)",
            pointHoverBorderColor: "rgba(220,220,220,1)",
            pointHoverBorderWidth: 2,
            pointRadius: 5,
            pointHitRadius: 10,
            data: [],
        }
    ]
};

const listedPrefixDataset = {
    label: "Route Changes (listed prefixes)",
    fill: false,
    lineTension: 0.1,
    backgroundColor: "rgba(15,151,3,0.4)",
    borderColor: "rgb(50,168,5)",
    borderCapStyle: 'butt',
    borderDash: [],
    borderDashOffset: 0.0,
    borderJoinStyle: 'miter',
    pointBorderColor: "rgb(75,192,81)",
    pointBackgroundColor: "#fff",
    pointBorderWidth: 1,
    pointHoverRadius: 5,
    pointHoverBackgroundColor: "rgb(75,192,102)",
    pointHoverBorderColor: "rgba(220,220,220,1)",
    pointHoverBorderWidth: 2,
    pointRadius: 5,
    pointHitRadius: 10,
    data: [],
};


const ctxFlapCount = document.getElementById('chartFlapCount').getContext('2d');
const ctxRoute = document.getElementById('chartRoute').getContext('2d');

const liveFlapChart = new Chart(ctxFlapCount, {
    type: "line",
    data: dataFlapCount,
    options: {
        maintainAspectRatio: false
    },
});


const liveRouteChart = new Chart(ctxRoute, {
    type: "line",
    data: dataRouteChange,
    options: {
        maintainAspectRatio: false,
        plugins: {
            tooltip: {
                callbacks: {
                    label: (context) => `${context.dataset.label}: ${context.parsed.y}/sec`,
                },
            },
        },
    },
});


async function updateCapabilities() {
    const response = await fetch("capabilities");
    const data = await response.json();
    const versionBox = document.getElementById("version");
    const infoBox = document.getElementById("info");
    versionBox.innerText = "FlapAlerted " + data.Version;
    if (data.UserParameters.RouteChangeCounter === 0) {
        infoBox.innerText = `Current settings: Displaying every BGP update received. Removing entries after ${data.UserParameters.FlapPeriod} seconds of inactivity.`;
    } else {
        dataRouteChange.datasets.push(listedPrefixDataset);
        infoBox.innerText = `Current settings: A route for a prefix needs to change at least ${data.UserParameters.RouteChangeCounter}  times in ${data.UserParameters.FlapPeriod} seconds and remain active for at least ${data.UserParameters.MinimumAge} seconds for it to be shown in the table.`;
    }
}

function addToChart(liveChart, point, unixTime, dataInterval) {
    const now = new Date(unixTime * 1000);
    const timeStamp = String(now.getHours()).padStart(2, '0') + ':' + String(now.getMinutes()).padStart(2, '0') +
        ":" + String(now.getSeconds()).padStart(2, '0');
    let shifted = false;
    for (let i = 0; i < point.length; i++) {
        if (liveChart.data.datasets[i] === undefined) {
            continue;
        }
        liveChart.data.datasets[i].data.push((point[i]/dataInterval).toFixed(2));

        if (liveChart.data.datasets[i].data.length > 50) {
            shifted = true;
            liveChart.data.datasets[i].data.shift();
        }
    }
    if (shifted) {
        liveChart.data.labels.shift();
    }
    liveChart.data.labels.push(timeStamp);
    liveChart.update();
}


window.onload = () => {
    updateCapabilities().catch((err) => {
        console.log(err);
    }).finally(() => {
        getStats();
        updateInfo();
        setInterval(updateInfo, 5000);
        document.getElementById("loadingScreen").style.display = 'none';
    });
};

function updateInfo() {
    fetch("flaps/active/compact").then(function (response) {
        return response.json();
    }).then(function (flapList) {
        handleConnectionLost(false, "updateInfo");
        flapList.sort((a, b) => b.TotalCount - a.TotalCount);

        let prefixTableHtml = '<thead><tr><th>Prefix</th><th>Duration</th><th>Route Changes</th></tr></thead><tbody>';
        for (let i = 0; i < flapList.length; i++) {
            let duration = toTimeElapsed(flapList[i].LastSeen - flapList[i].FirstSeen);
            prefixTableHtml += "<tr>";
            prefixTableHtml += "<td><a target=\"_blank\" href='analyze/?prefix=" + encodeURIComponent(flapList[i].Prefix) + "'>" + flapList[i].Prefix + "</a></td>";
            prefixTableHtml += "<td>" + duration + "</td>";
            prefixTableHtml += "<td>" + truncateRouteChanges(flapList[i].TotalCount) + "</td>";
            prefixTableHtml += "</tr>";
            if (i >= 100) {
                break;
            }
        }
        if (flapList.length === 0) {
            prefixTableHtml += '<tr><td colspan="3" class="centerText">Waiting for BGP flapping...</td></tr>';
        }
        prefixTableHtml += "</tbody>";
        document.getElementById("prefixTable").innerHTML = prefixTableHtml;
    }).catch(function (error) {
        handleConnectionLost(true, "updateInfo");
        console.log(error);
    });
}

function getStats() {
    const evtSource = new EventSource("flaps/statStream");
    const avgArray = [];
    evtSource.addEventListener("u", (event) => {
        try {
            const js = JSON.parse(event.data);

            addToChart(liveRouteChart, [js["Changes"], js["ListedChanges"]], js["Time"],5);
            addToChart(liveFlapChart, [js["Active"]], js["Time"],1);

            avgArray.push(js["Changes"]);
            if (avgArray.length > 50) {
                avgArray.shift();
            }

            let percentile = [...avgArray].sort((a, b) => a - b);
            percentile = percentile.slice(0, Math.ceil(percentile.length * 0.90));
            const sum = percentile.reduce((s, a) => s + a, 0);
            const avg = sum / percentile.length;
            gauge.refresh(avg / 5);

        } catch (err) {
            console.log(err);
        }
    });
    evtSource.onerror = (err) => {
        handleConnectionLost(true, "getStats");
        console.log(err);
    };
    evtSource.onopen = () => {
        handleConnectionLost(false, "getStats");
    };
}

function toTimeElapsed(secondsIn) {
    const secondsMinute = 60;
    const secondsHour = secondsMinute * 60;
    const secondsDay = secondsHour * 24;
    const days = Math.floor(secondsIn / secondsDay);
    const hours = Math.floor((secondsIn % secondsDay) / secondsHour).toString().padStart(2, '0');
    const minutes = Math.floor((secondsIn % secondsHour) / secondsMinute).toString().padStart(2, '0');
    const seconds = Math.floor(secondsIn % secondsMinute).toString().padStart(2, '0');
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
        return (+(routeChanges / million).toFixed(3)) + " million";
    } else if (routeChanges >= billion && routeChanges < trillion) {
        return (+(routeChanges / billion).toFixed(3)) + " billion";
    } else if (routeChanges >= trillion) {
        return (+(routeChanges / trillion).toFixed(2)) + " trillion";
    }
    return routeChanges;
}

let lostType = [];

function handleConnectionLost(lost, type) {
    if (lost) {
        if (lostType.indexOf(type) === -1) {
            lostType.push(type);
        }
        document.getElementById('connectionLost').style.display = 'block';
    } else {
        if (lostType.length !== 0) {
            lostType = lostType.filter(e => e !== type);
            if (lostType.length === 0) {
                document.getElementById('connectionLost').style.display = 'none';
            }
        }
    }
}