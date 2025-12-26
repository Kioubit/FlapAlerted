import "./chartjs/4.5.0/chart.umd.min.js";
import "./chartjs/chartjs-adapter-date-fns.bundle.min.js";

function getFetchOptions() {
    return {
        headers: {
            'X-AS': Math.floor(Date.now() / 1000).toString()
        }
    };
}

const noDataPlugin = {
    "id": "noDataToDisplay",
    "afterDraw": (chart) => {
        const hasData = chart.data.datasets.some(dataset =>
            Array.isArray(dataset.data) &&
            dataset.data.length > 0
        );

        if (!hasData) {
            const {ctx, width, height} = chart;
            chart.clear();
            ctx.save();
            ctx.textAlign = "center";
            ctx.textBaseline = "middle";
            ctx.font = 'bold 16px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto';
            ctx.fillStyle = "#999";
            ctx.fillText("Awaiting data to display", width / 2, height / 2);
            ctx.restore();
        }
    }
};

const ctxRouteCount = document.getElementById("chartRouteCount").getContext("2d");
const dataRouteChangeCount = {
    labels: [],
    datasets: [
        {
            label: "Route Changes per second",
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
        }
    ]
};


(function (){
    const ownURL = new URL(location.href);
    const prefix = ownURL.searchParams.get("prefix");
    const userDefined = ownURL.searchParams.get("userDefined") === "true";
    const timestamp = ownURL.searchParams.get("timestamp");
    if (!prefix) {
        document.getElementById("loader").style.display = "none";
        document.getElementById("loaderText").innerText = "Invalid link";
        return;
    }

    const historicalEndpoint = `../flaps/historical/prefix?prefix=${encodeURIComponent(prefix)}`;

    let prefixInfoEndpoint = `../flaps/prefix?prefix=${encodeURIComponent(prefix)}`;
    if (userDefined) {
        prefixInfoEndpoint = `../userDefined/prefix?prefix=${encodeURIComponent(prefix)}`;
    }
    if (timestamp) {
        prefixInfoEndpoint = `${historicalEndpoint}&timestamp=${timestamp}`;
    }
    fetch(prefixInfoEndpoint, getFetchOptions()).then((response) => response.json()).then(async (json) => {
        if (json === null && !timestamp) {
            const histResponse = await fetch(historicalEndpoint, getFetchOptions());
            const histJson = await histResponse.json();
            displayPrefix(histJson, false);
            return;
        }
        displayPrefix(json, userDefined);
    }).catch((error) => {
        document.getElementById("loader").style.display = "none";
        document.getElementById("loaderText").innerText = "An error occurred";
        alert("Network error");
        console.log(error);
    });
})();

function displayPrefix(json, userDefined) {
    if (json === null) {
        document.getElementById("loader").style.display = "none";
        document.getElementById("loaderText").innerText = "Prefix not found. The link may have expired";
        return;
    }
    
    // Determine if this is a historical event or a standard event
    let eventData = json;
    let reportTimestamp = Math.floor(Date.now() / 1000);
    const histDisplayEl = document.getElementById("historicalDisplay");

    if (json.Event && json.Meta) {
        // Handle historical wrapper format
        eventData = json.Event;
        reportTimestamp = json.Meta.Timestamp;

        if (histDisplayEl) {
            histDisplayEl.style.display = "block";
            histDisplayEl.innerText = `Historical Report (Event end date: ${timeConverter(reportTimestamp)})`;
        }
    } else {
        if (histDisplayEl) {
            histDisplayEl.style.display = "none";
        }
    }

    const pJson = eventData["PathHistory"];
    // pathMap contains path objects with the key being their first ASN
    const pathMap = new Map();

    if (pJson === null || pJson.length === 0) {
        document.getElementById("informationText2").innerText = "No path data is available yet. Try refreshing later.";
    } else {
        for (let i = 0; i < pJson.length; i++) {
            const firstAsn = pJson[i].Path[0];
            const targetArray = pathMap.get(firstAsn);
            if (!targetArray) {
                pathMap.set(firstAsn, [pJson[i]]);
            } else {
                targetArray.push(pJson[i]);
            }
        }
    }

    const htmlBundles = [];
    pathMap.forEach((value) => {
        // For each path group
        let elementHTML = "";
        let pathGroupTotalCount = 0;
        value.forEach((item) => {
            item.Count = item.ac + item.wc;
        });
        value.sort((a, b) => b.Count - a.Count);
        for (let c = 0; c < value.length; c++) {
            // For each path
            const count = value[c].Count;
            pathGroupTotalCount += count;
            elementHTML += `${count} (${value[c].ac}/${value[c].wc}) &nbsp;&nbsp;`;
            for (let d = 0; d < value[c].Path.length; d++) {
                // For each ASN in the path
                let singleAsn = value[c].Path[d].toString();
                elementHTML += `<span style='background-color: ${asnToColor(singleAsn)};'>&nbsp;${singleAsn.padStart(10, " ")}</span>`;
            }
            elementHTML += "<br>";
        }
        elementHTML += "<br>";
        const htmlBundle = {html: elementHTML, count: pathGroupTotalCount};
        htmlBundles.push(htmlBundle);
    });
    htmlBundles.sort((a, b) => b.count - a.count);

    let tableHtml = "";
    htmlBundles.forEach((bundle) => {
        tableHtml += bundle.html;
    });


    document.getElementById("pathTable").innerHTML = tableHtml;


    document.getElementById("prefixTitle").innerHTML = `Flap report for ${eventData.Prefix}`;
    document.getElementById("loader").style.display = "none";
    document.getElementById("loaderText").style.display = "none";


    document.getElementById("pathChangeDisplay").innerText = eventData.TotalPathChanges;
    document.getElementById("fistSeenDisplay").innerText = timeConverter(eventData.FirstSeen);
    document.getElementById("durationDisplay").innerText = toTimeElapsed(reportTimestamp - eventData.FirstSeen);


    document.getElementById("informationText1").style.display = "block";
    document.getElementById("informationText2").style.display = "block";

    if (!userDefined) {
        displayRateSecHistory(eventData.RateSecHistory, reportTimestamp)
    } else {
        document.getElementById("averageDisplay").innerText = "Not available for user-defined"``
    }

    const printButton = document.getElementById("printButton");
    if (printButton !== null) {
        printButton.onclick = () => {
            window.print();
        };
    }
}


function displayRateSecHistory(history, endTimestamp) {
    const dataIntervalSeconds = 60;
    document.getElementById("chartRouteCount-outerContainer").classList.remove("noDisplay");
    const RouteChangeChart = new Chart(ctxRouteCount, {
        type: "line",
        plugins: [noDataPlugin],
        data: dataRouteChangeCount,
        options: {
            animation: false,
            scales: {
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
    });
    window.addEventListener("beforeprint", () => {
        RouteChangeChart.resize(700, 150);
    });
    window.addEventListener("afterprint", () => {
        RouteChangeChart.resize();
    });

    if (history.length === 0) {
        return;
    }
    const t = endTimestamp * 1000;
    const labels = [];
    const data = [];
    for (let i = 1; i < history.length; i++) {
        // Timestamps are within an accuracy of about 60 seconds
        const ts = new Date(t - (1000 * dataIntervalSeconds * (history.length - i)));
        const timeStamp = `${String(ts.getHours()).padStart(2, "0")}:${String(ts.getMinutes()).padStart(2, "0")}:${String(ts.getSeconds()).padStart(2, "0")}`;
        labels.push(timeStamp);
        data.push(history[i]);
    }
    if (data.length === 0) {
        return;
    }

    const dataSum = data.reduce((s, a) => s + a, 0);
    const avg = ((dataSum / data.length)).toFixed(2);
    document.getElementById("averageDisplay").innerText = `${avg}/s during the last ${toTimeElapsed(data.length * dataIntervalSeconds)}`;

    RouteChangeChart.data.labels = labels;
    RouteChangeChart.data.datasets[0].data = data;
    RouteChangeChart.update();
}

function asnToColor(input) {
    let num = Number(input);
    // 1. Bit mixing logic
    // Bits are mixed so close numbers become different
    num ^= num >>> 16;
    num = Math.imul(num, 0x7feb352d);
    num ^= num >>> 15;
    num = Math.imul(num, 0x846ca68b);
    num ^= num >>> 16;

    // 2. Map to HSL
    // The scrambled number is mapped to 0-360 degrees
    const hue = Math.abs(num % 360);

    // 3. Vary saturation/lightness slightly based on other bits
    const sat = 65 + (Math.abs(num) % 30); // 65-95%
    const lig = 40 + (Math.abs(num) % 20); // 40-60%

    return `hsl(${hue}, ${sat}%, ${lig}%, 0.3)`;
}

function timeConverter(unixTimestamp) {
    function padTo2Digits(num) {
        return num.toString().padStart(2, "0");
    }

    const date = new Date(unixTimestamp * 1000);
    const hours = date.getHours();
    const minutes = date.getMinutes();
    const seconds = date.getSeconds();
    const time = `${padTo2Digits(hours)}:${padTo2Digits(minutes)}:${padTo2Digits(seconds)}`;

    const year = date.getFullYear();
    const month = padTo2Digits(date.getMonth() + 1);
    const day = padTo2Digits(date.getDate());

    return `${year}-${month}-${day} ${time}`;
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