window.onload = () => {
    display();
};

const ctxRouteCount = document.getElementById('chartRouteCount').getContext('2d');
const dataRouteChangeCount = {
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


function display() {
    const prefix = new URL(location.href).searchParams.get("prefix");
    if (prefix == null) {
        alert("Invalid link");
        return;
    }
    fetch("../flaps/prefix?prefix=" + encodeURIComponent(prefix)).then(function (response) {
        return response.json();
    }).then(function (json) {
        if (json === null) {
            document.getElementById("loader").style.display = "none";
            document.getElementById("loaderText").innerText = "Prefix not found. The link may have expired";
            return;
        }

        const pJson = json["Paths"];

        let obj = [];
        for (let i = 0; i < pJson.length; i++) {
            let firstAsn = pJson[i].Path.Asn[0];
            if (obj[firstAsn] == null) {
                obj[firstAsn] = [pJson[i]];
            } else {
                obj[firstAsn].push(pJson[i]);
            }
        }

        let htmlBundles = [];
        Object.entries(obj).forEach(([_, value]) => {
            let elementHTML = "";
            let totalCount = 0;
            for (let c = 0; c < value.length; c++) {
                let count = value[c].Count;
                totalCount += count;
                elementHTML += count + "&nbsp;&nbsp;";
                for (let d = 0; d < value[c].Path.Asn.length; d++) {
                    let sa = value[c].Path.Asn[d].toString();
                    let saLen = sa.length;
                    let gap = " ";
                    while (saLen < 10) {
                        gap = gap + "&nbsp;";
                        saLen++;
                    }
                    let hexColor = stringToColor(sa);
                    let r = parseInt(hexColor.slice(1, 3), 16);
                    let g = parseInt(hexColor.slice(3, 5), 16);
                    let b = parseInt(hexColor.slice(5, 7), 16);
                    elementHTML += `<span style='background-color: rgba(${r},${g},${b},0.3);'>${gap}${sa}</span>`;
                }
                elementHTML += "<br>";
            }
            elementHTML += "<br>";
            let htmlBundle = {html: elementHTML, count: totalCount};
            htmlBundles.push(htmlBundle);
        });
        htmlBundles.sort((a, b) => b.count - a.count);

        let tableHtml = '';
        htmlBundles.forEach((bundle) => {
            tableHtml += bundle.html;
        });

        if (tableHtml === "") {
            document.getElementById("pathTable").innerHTML = "No path data is available yet. Try refreshing later.";
            (async () => {
                try {
                    const response = await fetch("capabilities");
                    const data = await response.json();
                    if (!data["UserParameters"]["KeepPathInfo"]) {
                        document.getElementById("pathTable").innerHTML = "This instance has been configured to not keep path information." +
                            " THe analysis tool is not available as a result";
                    }
                } catch (e) {
                    console.log(e);
                }
            })();
        } else {
            document.getElementById("pathTable").innerHTML = tableHtml;
        }

        document.getElementById("prefixTitle").innerHTML = "Flap report for " + prefix;
        document.getElementById("loader").style.display = "none";
        document.getElementById("loaderText").style.display = "none";


        document.getElementById("pathChangeDisplay").innerText = json.TotalCount;
        document.getElementById("fistSeenDisplay").innerText = timeConverter(json.FirstSeen);
        document.getElementById("lastSeenDisplay").innerText = timeConverter(json.LastSeen);
        document.getElementById("durationDisplay").innerText = toTimeElapsed(json.LastSeen - json.FirstSeen);

        document.getElementById("informationText1").style.display = "block";
        document.getElementById("informationText2").style.display = "block";
        document.getElementById("printButton").onclick = function () {
            window.print();
        };
    }).catch(function (error) {
        alert("Network error");
        console.log(error);
    });


    fetch("../flaps/active/history?cidr=" + prefix).then(function (response) {
        return response.json();
    }).then(function (json) {
        const dataIntervalSeconds = 10
        if (json.length === 0) {
            return;
        }
        const RouteChangeChart = new Chart(ctxRouteCount, {
            type: "line",
            data: dataRouteChangeCount,
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
        window.addEventListener('beforeprint', () => {
            RouteChangeChart.resize(700, 150);
        });
        window.addEventListener('afterprint', () => {
            RouteChangeChart.resize();
        });

        const t = Date.now();
        const labels = [];
        const data = [];
        let previousValue = json[0];
        for (let i = 1; i < json.length; i++) {
            // Timestamps are within an accuracy of about 10 seconds
            const ts = new Date(t - (10000 * (json.length - i)));
            const timeStamp = String(ts.getHours()).padStart(2, '0') + ':' +
                String(ts.getMinutes()).padStart(2, '0') + ":" + String(ts.getSeconds()).padStart(2, '0');
            labels.push(timeStamp);
            data.push(((json[i] - previousValue)/dataIntervalSeconds));
            previousValue = json[i];
        }
        if (data.length === 0 ) {
            return;
        }

        const dataSum = data.reduce((s,a) => s + a , 0);
        const avg = ((dataSum/data.length)).toFixed(2);
        document.getElementById("averageDisplay").innerText = `${avg}/s during the last ${toTimeElapsed(data.length*dataIntervalSeconds)}`;

        RouteChangeChart.data.labels = labels;
        RouteChangeChart.data.datasets[0].data = data;
        RouteChangeChart.update();
    }).catch(function (error) {
        alert("Network error");
        console.log(error);
    });
}


function stringToColor(str) {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        hash = str.charCodeAt(i) + ((hash << 5) - hash);
    }
    let colour = '#';
    for (let i = 0; i < 3; i++) {
        let value = (hash >> (i * 8)) & 0xFF;
        let rawColour = '00' + value.toString(16);
        colour += rawColour.substring(rawColour.length - 2);
    }
    return colour;
}

function timeConverter(unixTimestamp) {
    function padTo2Digits(num) {
        return num.toString().padStart(2, '0');
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