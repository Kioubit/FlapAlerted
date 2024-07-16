function updateCapabilities() {
    let getCapabilitiesFunction = async () => {
        const response = await fetch("capabilities");
        return await response.json();
    }
    const versionBox = document.getElementById("version");
    const infoBox = document.getElementById("info");
    getCapabilitiesFunction().then((data) => {
        versionBox.innerText =  "FlapAlerted " + data.Version;
        if (data.UserParameters.NotifyTarget === 0) {
            infoBox.innerText = "Current settings: Displaying every BGP update received. Removing entries after "+ data.UserParameters.FlapPeriod + " seconds of inactivity.";
        } else {
            infoBox.innerText = "Current settings: A route for a prefix needs to change at least " + data.UserParameters.NotifyTarget + " times in " + data.UserParameters.FlapPeriod + " seconds for it to be shown in the table.";
        }
    }).catch((err) => {
        versionBox.innerHTML += "N/A";
        console.log(err);
    });
}

let gauge = new JustGage({
    id: "justgage",
    value: 0,
    min: 0,
    max: 1000,
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

const ctxFlapCount = document.getElementById('chartFlapCount').getContext('2d');
const ctxRoute = document.getElementById('chartRoute').getContext('2d');

let dataFlapCount = {
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


let dataRouteChange = {
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



let liveFlapChart = new Chart(ctxFlapCount, {
    type: "line",
    data: dataFlapCount,
    options: {
        maintainAspectRatio: false
    },
})


let liveRouteChart = new Chart(ctxRoute, {
    type: "line",
    data: dataRouteChange,
    options: {
        maintainAspectRatio: false
    },
})

function addToChart(liveChart, point, unixTime) {
    const now = new Date(unixTime * 1000);
    const timeStamp = String(now.getHours()).padStart(2, '0') + ':' + String(now.getMinutes()).padStart(2, '0')
        + ":" + String(now.getSeconds()).padStart(2, '0');

    liveChart.data.labels.push(timeStamp);
    liveChart.data.datasets[0].data.push(point)

    if (liveChart.data.datasets[0].data.length > 50) {
        liveChart.data.labels.shift();
        liveChart.data.datasets[0].data.shift();
    }
    liveChart.update();
}



window.onload = () => {
    updateCapabilities();
    updateInfo();
    setInterval(updateInfo, 5000);
    getStats();
    document.getElementById("loadingScreen").style.display = 'none';
}

function updateInfo() {
    fetch("flaps/active/compact").then(function (response) {
        return response.json();
    }).then(function (flapList) {
        displayConnectionLost(false, "updateInfo");
        flapList.sort(function compareFn(a, b) {
            if (a.TotalCount > b.TotalCount) {
                return -1;
            } else {
                return 1;
            }
        });

        let prefixTableHtml = '<table id="prefixTable"><thead><tr><th>Prefix</th><th>Duration</th><th>Route Changes</th></tr></thead><tbody>';
        for (let i = 0; i < flapList.length; i++) {
            let duration = toTimeElapsed(flapList[i].LastSeen - flapList[i].FirstSeen);
            prefixTableHtml += "<tr>";
            prefixTableHtml += "<td><a target=\"_blank\" href='analyze/?prefix=" + encodeURIComponent(flapList[i].Prefix) +"'>" + flapList[i].Prefix + "</a></td>";
            prefixTableHtml += "<td>" + duration + "</td>";
            prefixTableHtml += "<td>" + flapList[i].TotalCount + "</td>";
            prefixTableHtml += "</tr>";
        }
        if (flapList.length === 0) {
            prefixTableHtml += '<tr><td colspan="3" class="centerText">Waiting for BGP flapping...</td></tr>';
        }
        prefixTableHtml += "</tbody></table>";
        document.getElementById("prefixTableBox").innerHTML = prefixTableHtml;

    }).catch(function (error) {
        displayConnectionLost(true, "updateInfo");
        console.log(error);
    });
}

function getStats() {
    const evtSource = new EventSource("flaps/statStream");
    const avgArray = [];
    evtSource.addEventListener("u", (event) => {
        try {
            const js = JSON.parse(event.data)

            addToChart(liveRouteChart, js["Changes"], js["Time"]);
            addToChart(liveFlapChart, js["Active"], js["Time"]);

            avgArray.push(js["Changes"])
            if (avgArray.length > 50) {
                avgArray.shift()
            }
            const sum = avgArray.reduce((s, a) => s + a, 0)
            const avg = sum/avgArray.length;
            gauge.refresh(avg);

        } catch(err) {
            console.log(err);
        }
    });
    evtSource.onerror = (err) => {
        displayConnectionLost(true, "getStats");
        console.log(err)
    };
    evtSource.onopen = () => {
        displayConnectionLost(false, "getStats");
    };
}

function toTimeElapsed(seconds) {
    let date = new Date(null);
    date.setSeconds(seconds);
    return date.toISOString().slice(11, 19);
}

let lostType = [];
function displayConnectionLost(lost, type) {
    if (lost) {
        if (lostType.indexOf(type) === -1) {
            lostType.push(type)
        }
        document.getElementById('connectionLost').style.display = 'block';
    } else {
        lostType = lostType.filter(e => e !== type);
        if (lostType.length === 0) {
            document.getElementById('connectionLost').style.display = 'none';
        }
    }
}