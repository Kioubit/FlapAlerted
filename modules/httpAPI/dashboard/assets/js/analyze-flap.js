window.addEventListener('load', function () {
    display();
});

function display() {
    fetch("../flaps/active").then(function (response) {
        return response.json();
    }).then(function (json) {
        let pJson;
        let prefix = new URL(location.href).searchParams.get("prefix");
        if (prefix == null) {
            alert("Invalid link");
            return;
        }

        for (let i = 0; i < json.length; i++) {
            if (json[i].Prefix === prefix) {
                pJson = json[i].Paths
                break;
            }
        }

        if (pJson == null) {
            alert("Prefix not found");
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
        document.getElementById("prefixTitle").innerHTML = "Flap analysis for " + prefix;
        document.getElementById("loader").style.display = "none";
        document.getElementById("loaderText").style.display = "none";
    }).catch(function (error) {
        alert("Network error");
        console.log(error);
    });

    document.getElementById("informationText").style.display = "block";
    document.getElementById("printButton").onclick = function () {
        window.print();
    }
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
