window.addEventListener('load', function () {
    display();
});

const ctxRouteCount = document.getElementById('chartRouteCount').getContext('2d');
const dataRouteChangeCount = {
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



function display() {
    const prefix = new URL(location.href).searchParams.get("prefix");
    let targetJson;
    fetch("../flaps/active").then(function (response) {
        return response.json();
    }).then(function (json) {
        let pJson;
        if (prefix == null) {
            alert("Invalid link");
            return;
        }

        for (let i = 0; i < json.length; i++) {
            if (json[i].Prefix === prefix) {
                targetJson = json[i];
                pJson = json[i].Paths;
                break;
            }
        }

        if (pJson == null) {
            document.getElementById("loader").style.display = "none";
            document.getElementById("loaderText").innerText = "Prefix not found. The link may have expired";
            //alert("Prefix not found");
            return;
        }

        if (pJson.length === 0) {
            alert("The analysis feature is not available as the instance has been configured to not keep path information")
        }

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

        for (const key in obj) {
            let elementHTML = "";
            let totalCount = 0;
            for (let c = 0; c < obj[key].length; c++) {
                let count = obj[key][c].Count;
                totalCount += count;
                elementHTML += count + "&nbsp;&nbsp;";
                for (let d = 0; d < obj[key][c].Path.Asn.length; d++) {
                    let sa = obj[key][c].Path.Asn[d].toString();
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
                    elementHTML += "<span style='background-color: rgba(" + r + "," + g + "," + b + "," + "0.3');>" + gap + sa + "</span>";
                }
                elementHTML += "<br>";
            }
            elementHTML += "<br>";
            let htmlBundle = {html: elementHTML, count: totalCount};
            htmlBundles.push(htmlBundle);
        }
        htmlBundles.sort((a,b)=> {
            if (a.count < b.count) {
                return 1;
            }  else if (a.count > b.count) {
                return -1;
            }
            return 0;
        });
        let tableHtml = '';
        htmlBundles.forEach((bundle) => {
            tableHtml += bundle.html;
        })

        document.getElementById("pathTable").innerHTML = tableHtml;
        document.getElementById("prefixTitle").innerHTML = "Flap report for " + prefix;
        document.getElementById("loader").style.display = "none";
        document.getElementById("loaderText").style.display = "none";


        document.getElementById("pathChangeDisplay").innerText = targetJson.TotalCount;
        document.getElementById("fistSeenDisplay").innerText = timeConverter(targetJson.FirstSeen);
        document.getElementById("lastSeenDisplay").innerText = timeConverter(targetJson.LastSeen);
        document.getElementById("durationDisplay").innerText = toTimeElapsed(targetJson.LastSeen - targetJson.FirstSeen);

        document.getElementById("informationText1").style.display = "block";
        document.getElementById("informationText2").style.display = "block";
        document.getElementById("printButton").onclick = function () {
            window.print();
        }
    }).catch(function (error) {
        alert("Network error");
        console.log(error);
    });


    fetch("../flaps/active/history?cidr=" + prefix).then(function (response) {
        return response.json();
    }).then(function (json) {
        if (json.length === 0) {
            return;
        }
        const RouteCountChart = new Chart(ctxRouteCount, {
            type: "line",
            data: dataRouteChangeCount,
            options: {
                maintainAspectRatio: false
            },
        })

        let labels = [];
        for (let i = 0; i < json.length; i++) {
            labels.push(i);
        }
        RouteCountChart.data.labels = labels;
        RouteCountChart.data.datasets[0].data = json;
        RouteCountChart.update();

        window.addEventListener('beforeprint', (event) => {
            RouteCountChart.resize();
        });
        window.addEventListener('afterprint', () => {
            RouteCountChart.resize();
        });
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
        colour += rawColour.substring(rawColour.length-2);
    }
    return colour;
}

function timeConverter(unixTimestamp){
    const date = new Date(unixTimestamp * 1000);
    const hours = date.getHours();
    const minutes = date.getMinutes();
    const seconds = date.getSeconds();
    const time = `${padTo2Digits(hours)}:${padTo2Digits(minutes)}:${padTo2Digits(
        seconds,
    )}`;

    const year = date.getFullYear();
    const month = padTo2Digits(date.getMonth() + 1);
    const day = padTo2Digits(date.getDate());

    return `${year}-${month}-${day} ${time}`;
}

function padTo2Digits(num) {
    return num.toString().padStart(2, '0');
}

function toTimeElapsed(seconds) {
    let date = new Date(null);
    date.setSeconds(seconds);
    return date.toISOString().slice(11, 19);
}