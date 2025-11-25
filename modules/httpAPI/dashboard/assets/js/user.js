import "./chartjs/4.5.0/chart.umd.min.js";
import "./chartjs/chartjs-adapter-date-fns.bundle.min.js";

const dataRouteChange = {
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
}

const ctxRouteChange = document.getElementById("chartRouteChange").getContext("2d");
const liveRouteChart = new Chart(
    ctxRouteChange,
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

let fatalErrorReported = false;
(function start() {
    const loadingScreen = document.getElementById("loadingScreen");
    const totalChangesDiv = document.getElementById("totalChanges");
    const prefixDisplay = document.getElementById("prefix");
    const prefixLink = document.getElementById("prefixLink");
    const errorDisplay = document.getElementById("error");
    const noBGPFeeds = document.getElementById("noBGPFeeds");
    const mainInfoDiv = document.getElementById("mainInfo");

    const prefix = new URL(location.href).searchParams.get("prefix").trim();
    if (prefix === null) {
        loadingScreen.style.display = "none";
        mainInfoDiv.style.display = "none";
        errorDisplay.innerText = "Prefix not provided";
        errorDisplay.style.display = "block";
        return
    }

    const evtSource = new EventSource(`../userDefined/subscribe?prefix=${encodeURIComponent(prefix)}`);

    let lastValue = null;
    evtSource.addEventListener("e", (event) => {
        errorDisplay.innerText = event.data;
        errorDisplay.style.display = "block";
        fatalErrorReported = true;
        mainInfoDiv.style.display = "none";
        evtSource.close();
    });
    evtSource.addEventListener("valid", (_) => {
        lastValue = null;
        prefixDisplay.innerText = prefix;
        prefixLink.href = `../analyze/?prefix=${prefix}&userDefined=true`;
    })
    evtSource.addEventListener("u", (event) => {
        const js = JSON.parse(event.data);
        const firstRun = lastValue === null;
        if (firstRun) {
            lastValue = 0;
        }
        const difference = js.Count - lastValue;
        lastValue = js.Count;

        if (js.Sessions === 0) {
            noBGPFeeds.style.display = "block";
        } else {
            noBGPFeeds.style.display = "none";
        }

        totalChangesDiv.innerText = `Total path changes: ${js.Count}`;

        if (firstRun) {
            return;
        }

        liveRouteChart.data.datasets[0].data.push(difference/5);
        if (liveRouteChart.data.datasets[0].data.length > 50) {
            liveRouteChart.data.datasets[0].data.shift();
            liveRouteChart.data.labels.shift()
        }
        liveRouteChart.data.labels.push(Date.now());
        liveRouteChart.update();
    });
    evtSource.onerror = (err) => {
        loadingScreen.style.display = "none";
        handleConnectionLost(true);
        console.log(err);
    };
    evtSource.onopen = () => {
        loadingScreen.style.display = "none";
        handleConnectionLost(false);
    };
})();

function handleConnectionLost(lost) {
    if (fatalErrorReported) {
        return;
    }
    if (lost) {
        document.getElementById("connectionLost").style.display = "block";
    } else {
        document.getElementById("connectionLost").style.display = "none";
    }
}

