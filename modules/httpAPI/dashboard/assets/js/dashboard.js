let getCapabilitiesFunction = async () => {
    const response = await fetch("../capabilities");
    return await response.json();
}
const versionBox = document.getElementById("version");
const infoBox = document.getElementById("info");
const pathListLink = document.getElementById("PathListLink");
versionBox.innerHTML = "Version: FlapAlertedPro ";
getCapabilitiesFunction().then((data) => {
    versionBox.innerHTML += data.Version;
    if (!data.UserParameters.KeepPathInfo) {
        pathListLink.href = '';
        pathListLink.innerText += " (Disabled by the administrator)";
        pathListLink.classList.add("disabledLink");
    }
    if (data.UserParameters.NotifyTarget === 0) {
        infoBox.innerText = "Current settings: Displaying every BGP Update received (This does not indicate flapping). Removing entries after "+ data.UserParameters.FlapPeriod + " seconds of inactivity.";
        document.getElementById("dataInfo").innerText = "Live Data -- All BGP Updates";
    } else {
        infoBox.innerText = "Current settings: Detecting a flap if a route changes at least " + data.UserParameters.NotifyTarget + " times in " + data.UserParameters.FlapPeriod + " seconds.";
        document.getElementById("dataInfo").innerText = "Live Data";
    }
}).catch((err) => {
    versionBox.innerHTML += "N/A";
});


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
            hi: 10
        },
        {
            color: "#f7bc08",
            lo: 11,
            hi: 50
        },
        {
            color: "#ff3b30",
            lo: 51,
            hi: 100
        }]
    }
});


const ctxFlapCount = document.getElementById('chartFlapCount').getContext('2d');
const ctxRouteCount = document.getElementById('chartRouteCount').getContext('2d');
const ctxRoute = document.getElementById('chartRoute').getContext('2d');

const emptyChartPlugin = {
    id: 'emptyChart',
    afterDraw(chart, args, options) {
        const { datasets } = chart.data;
        let hasData = false;

        for (let dataset of datasets) {
            if (dataset.data.length > 0 && dataset.data.some(item => item !== 0)) {
                hasData = true;
                break;
            }
        }

        if (!hasData) {
            const { chartArea: { left, top, right, bottom }, ctx } = chart;
            const centerX = (left + right) / 2;
            const centerY = (top + bottom) / 2;

            chart.clear();
            ctx.save();
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('Waiting for BGP flapping...', centerX, centerY);
            ctx.restore();
        }
    }
};

let dataFlapCount = {
    labels: [],
    datasets: [
        {
            label: "Count of actively flapping prefixes",
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


let dataRouteChangeCount = {
    labels: [],
    datasets: [
        {
            label: "Route Change count",
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
    plugins: [emptyChartPlugin],
    options: {
        maintainAspectRatio: false
    },
})

let liveRouteCountChart = new Chart(ctxRouteCount, {
    type: "line",
    data: dataRouteChangeCount,
    plugins: [emptyChartPlugin],
    options: {
        maintainAspectRatio: false
    },
})

let liveRouteChart = new Chart(ctxRoute, {
    type: "line",
    data: dataRouteChange,
    plugins: [emptyChartPlugin],
    options: {
        maintainAspectRatio: false
    },
})

function addToChart(liveChart, point) {
    const now = new Date();
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


let flapList = [];
let avgDerivArr = [];
let newCount = 0;
let oldCount = 0;
updateInfo();
setInterval(updateInfo, 5000);
let firstPass = true;
function updateInfo() {
    fetch("../flaps/active/compact").then(function (response) {
        return response.json();
    }).then(function (json) {
        document.getElementById('connectionLost').style.display = 'none';
        for (const key in json) {
            let prefix = json[key].Prefix;
            let index = flapList.findIndex((obj => obj.Prefix === prefix));
            if (index === -1) {
                flapList.push(json[key]);
                continue;
            }

            //reuse
            if (flapList[index].FirstSeen < json[key].FirstSeen) {
                if ('reuse' in flapList[index]) {
                    flapList[index].reuse += flapList[index].TotalCount;
                } else {
                    flapList[index].reuse = flapList[index].TotalCount
                }
                flapList[index].FirstSeen = json[key].FirstSeen;
            }

            if ('reuse' in flapList[index]) {
                flapList[index].TotalCount = json[key].TotalCount;
                flapList[index].LastSeen = json[key].LastSeen;
            } else {
                flapList[index] = json[key];
            }
        }

        let index = flapList.length
        while (index--) {
            let searchPrefix = flapList[index].Prefix;
            let jsonIndex = json.findIndex((obj => obj.Prefix === searchPrefix));
            if (jsonIndex === -1) {
                oldCount = oldCount - flapList[index].TotalCount;
                if ('reuse' in flapList[index]) {
                    oldCount = oldCount - flapList[index].reuse;
                }
                flapList.splice(index, 1);
            } else {
                if ('reuse' in flapList[index]) {
                    newCount = newCount + flapList[index].TotalCount + flapList[index].reuse;
                } else {
                    newCount = newCount + flapList[index].TotalCount;
                }
            }
        }
        let difference = newCount - oldCount;
        oldCount = newCount;
        newCount = 0;

        if (firstPass) {
            firstPass = false;
            difference = 0;
        }

        avgDerivArr.push(difference);
        if (avgDerivArr.length > 50) {
            avgDerivArr.splice(0, 1);
        }
        let avgDeriv = 0
        for (let i = 0; i < avgDerivArr.length; i++) {
            avgDeriv = avgDeriv + avgDerivArr[i];
        }
        avgDeriv = avgDeriv / avgDerivArr.length

        addToChart(liveRouteCountChart, oldCount);
        addToChart(liveRouteChart, difference);
        addToChart(liveFlapChart, flapList.length);
        gauge.refresh(avgDeriv);

        flapList.sort(function compareFn(a, b) {
            if (a.TotalCount > b.TotalCount) {
                return -1;
            } else {
                return 1;
            }
        });

        let prefixTableHtml = '<table id="prefixTable"><tr><th>Prefix</th><th>Seconds</th><th>Route Changes</th></tr>';
        for (let i = 0; i < flapList.length; i++) {
            let duration = flapList[i].LastSeen - flapList[i].FirstSeen;
            prefixTableHtml += "<tr>";
            prefixTableHtml += "<td><a target=\"_blank\" href='analyze/?prefix=" + encodeURIComponent(flapList[i].Prefix) +"'>" + flapList[i].Prefix + "</a></td>";
            prefixTableHtml += "<td>" + duration + "</td>";
            prefixTableHtml += "<td>" + flapList[i].TotalCount + "</td>";
            prefixTableHtml += "</tr>";
        }
        if (flapList.length === 0) {
            prefixTableHtml += '<tr><td colspan="3" class="centerText">Waiting for BGP flapping...</td></tr>';
        }
        prefixTableHtml += "</table>";
        document.getElementById("prefixTableBox").innerHTML = prefixTableHtml;

    }).catch(function (error) {
        firstPass = true;
        document.getElementById('connectionLost').style.display = 'block';
        console.log(error);
    });

}
