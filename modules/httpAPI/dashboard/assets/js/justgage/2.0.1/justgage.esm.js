// JustGage v2.0.1 - Modern ES6+ SVG Gauges
// Zero dependencies, native SVG rendering

var __defProp = Object.defineProperty;
var __name = (target, value) => __defProp(target, "name", { value, configurable: true });

// src/utils/helpers.js
function isUndefined(v) {
  return v === null || v === void 0;
}
__name(isUndefined, "isUndefined");
function isNumber(n) {
  return n !== null && n !== void 0 && !isNaN(n);
}
__name(isNumber, "isNumber");
function extend(out, ...sources) {
  out = out || {};
  for (const source of sources) {
    if (!source) {
      continue;
    }
    for (const key in source) {
      if (Object.prototype.hasOwnProperty.call(source, key)) {
        out[key] = source[key];
      }
    }
  }
  return out;
}
__name(extend, "extend");
function uuid() {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = Math.random() * 16 | 0;
    const v = c === "x" ? r : r & 3 | 8;
    return v.toString(16);
  });
}
__name(uuid, "uuid");
function kvLookup(key, tableA, tableB, defVal, dataType) {
  let val = defVal;
  let canConvert = false;
  if (!isUndefined(key)) {
    if (!isUndefined(tableB) && typeof tableB === "object" && key in tableB) {
      val = tableB[key];
      canConvert = true;
    } else if (!isUndefined(tableA) && typeof tableA === "object" && key in tableA) {
      val = tableA[key];
      canConvert = true;
    } else {
      val = defVal;
    }
    if (canConvert && !isUndefined(dataType)) {
      switch (dataType) {
        case "int":
          val = parseInt(val, 10);
          break;
        case "float":
          val = parseFloat(val);
          break;
        default:
          break;
      }
    }
  }
  return val;
}
__name(kvLookup, "kvLookup");

// src/core/config.js
var GAUGE_WIDTH_DIVISOR = 20 / 3;
var DEFAULT_CONFIG = {
  value: 0,
  min: 0,
  max: 100,
  reverse: false,
  gaugeWidthScale: 1,
  gaugeColor: "#edebeb",
  label: "",
  valueFontColor: "#010101",
  valueFontFamily: "Arial",
  labelFontColor: "#b3b3b3",
  labelFontFamily: "Arial",
  symbol: "",
  shadowOpacity: 0.2,
  shadowSize: 5,
  shadowVerticalOffset: 3,
  levelColors: ["#a9d70b", "#f9c802", "#ff0000"],
  startAnimationTime: 700,
  startAnimationType: ">",
  refreshAnimationTime: 700,
  refreshAnimationType: ">",
  donutStartAngle: 90,
  valueMinFontSize: 16,
  labelMinFontSize: 10,
  minLabelMinFontSize: 10,
  maxLabelMinFontSize: 10,
  titleMinFontSize: 10,
  hideValue: false,
  hideMinMax: false,
  showMinMax: true,
  showInnerShadow: false,
  humanFriendly: false,
  humanFriendlyDecimal: 0,
  noGradient: false,
  donut: false,
  differential: false,
  relativeGaugeSize: false,
  counter: false,
  decimals: 0,
  customSectors: {},
  formatNumber: false,
  pointer: false,
  pointerOptions: {},
  displayRemaining: false,
  targetLine: null,
  targetLineColor: "#000000",
  targetLineWidth: 1.5,
  textRenderer: null,
  onAnimationEnd: null,
  showSectorColors: false,
  minTxt: false,
  maxTxt: false,
  defaults: false,
  parentNode: null,
  width: null,
  height: null,
  title: "",
  titleFontColor: "#999999",
  titleFontFamily: "Arial",
  titleFontWeight: "bold",
  titlePosition: "above",
  // above|below
  valueFontWeight: "normal",
  labelFontWeight: "normal"
};
function createConfig(config, dataset = {}) {
  if (isUndefined(config)) {
    throw new Error("JustGage: Configuration object is required");
  }
  const { defaults, ...restConfig } = config;
  if (defaults) {
    config = extend({}, defaults, restConfig);
  }
  const processedConfig = {
    // Generate unique class ID for styling
    classId: uuid(),
    // Core identification
    id: config.id,
    parentNode: kvLookup("parentNode", config, dataset, null),
    // Dimensions
    width: kvLookup("width", config, dataset, DEFAULT_CONFIG.width),
    height: kvLookup("height", config, dataset, DEFAULT_CONFIG.height),
    // Value settings
    value: kvLookup("value", config, dataset, DEFAULT_CONFIG.value, "float"),
    min: kvLookup("min", config, dataset, DEFAULT_CONFIG.min, "float"),
    max: kvLookup("max", config, dataset, DEFAULT_CONFIG.max, "float"),
    minTxt: kvLookup("minTxt", config, dataset, DEFAULT_CONFIG.minTxt),
    maxTxt: kvLookup("maxTxt", config, dataset, DEFAULT_CONFIG.maxTxt),
    reverse: kvLookup("reverse", config, dataset, DEFAULT_CONFIG.reverse),
    // Display settings
    symbol: kvLookup("symbol", config, dataset, DEFAULT_CONFIG.symbol),
    decimals: kvLookup("decimals", config, dataset, DEFAULT_CONFIG.decimals),
    counter: kvLookup("counter", config, dataset, DEFAULT_CONFIG.counter),
    hideValue: kvLookup("hideValue", config, dataset, DEFAULT_CONFIG.hideValue),
    hideMinMax: kvLookup("hideMinMax", config, dataset, DEFAULT_CONFIG.hideMinMax),
    showMinMax: kvLookup("showMinMax", config, dataset, DEFAULT_CONFIG.showMinMax),
    // Fonts and colors
    valueFontColor: kvLookup("valueFontColor", config, dataset, DEFAULT_CONFIG.valueFontColor),
    valueFontFamily: kvLookup("valueFontFamily", config, dataset, DEFAULT_CONFIG.valueFontFamily),
    labelFontColor: kvLookup("labelFontColor", config, dataset, DEFAULT_CONFIG.labelFontColor),
    labelFontFamily: kvLookup("labelFontFamily", config, dataset, DEFAULT_CONFIG.labelFontFamily),
    // Font sizes
    valueMinFontSize: kvLookup(
      "valueMinFontSize",
      config,
      dataset,
      DEFAULT_CONFIG.valueMinFontSize
    ),
    labelMinFontSize: kvLookup(
      "labelMinFontSize",
      config,
      dataset,
      DEFAULT_CONFIG.labelMinFontSize
    ),
    minLabelMinFontSize: kvLookup(
      "minLabelMinFontSize",
      config,
      dataset,
      DEFAULT_CONFIG.minLabelMinFontSize
    ),
    maxLabelMinFontSize: kvLookup(
      "maxLabelMinFontSize",
      config,
      dataset,
      DEFAULT_CONFIG.maxLabelMinFontSize
    ),
    titleMinFontSize: kvLookup(
      "titleMinFontSize",
      config,
      dataset,
      DEFAULT_CONFIG.titleMinFontSize
    ),
    // Gauge appearance
    gaugeWidthScale: kvLookup("gaugeWidthScale", config, dataset, DEFAULT_CONFIG.gaugeWidthScale),
    gaugeColor: kvLookup("gaugeColor", config, dataset, DEFAULT_CONFIG.gaugeColor),
    levelColors: kvLookup("levelColors", config, dataset, DEFAULT_CONFIG.levelColors),
    noGradient: kvLookup("noGradient", config, dataset, DEFAULT_CONFIG.noGradient),
    // Shadow settings
    shadowOpacity: kvLookup("shadowOpacity", config, dataset, DEFAULT_CONFIG.shadowOpacity),
    shadowSize: kvLookup("shadowSize", config, dataset, DEFAULT_CONFIG.shadowSize),
    shadowVerticalOffset: kvLookup(
      "shadowVerticalOffset",
      config,
      dataset,
      DEFAULT_CONFIG.shadowVerticalOffset
    ),
    showInnerShadow: kvLookup("showInnerShadow", config, dataset, DEFAULT_CONFIG.showInnerShadow),
    // Animation settings
    startAnimationTime: kvLookup(
      "startAnimationTime",
      config,
      dataset,
      DEFAULT_CONFIG.startAnimationTime
    ),
    startAnimationType: kvLookup(
      "startAnimationType",
      config,
      dataset,
      DEFAULT_CONFIG.startAnimationType
    ),
    refreshAnimationTime: kvLookup(
      "refreshAnimationTime",
      config,
      dataset,
      DEFAULT_CONFIG.refreshAnimationTime
    ),
    refreshAnimationType: kvLookup(
      "refreshAnimationType",
      config,
      dataset,
      DEFAULT_CONFIG.refreshAnimationType
    ),
    // Gauge types
    donut: kvLookup("donut", config, dataset, DEFAULT_CONFIG.donut),
    donutStartAngle: kvLookup("donutStartAngle", config, dataset, DEFAULT_CONFIG.donutStartAngle),
    differential: kvLookup("differential", config, dataset, DEFAULT_CONFIG.differential),
    relativeGaugeSize: kvLookup(
      "relativeGaugeSize",
      config,
      dataset,
      DEFAULT_CONFIG.relativeGaugeSize
    ),
    // Advanced features
    customSectors: kvLookup("customSectors", config, dataset, DEFAULT_CONFIG.customSectors),
    pointer: kvLookup("pointer", config, dataset, DEFAULT_CONFIG.pointer),
    pointerOptions: kvLookup("pointerOptions", config, dataset, DEFAULT_CONFIG.pointerOptions),
    targetLine: kvLookup("targetLine", config, dataset, DEFAULT_CONFIG.targetLine, "float"),
    targetLineColor: kvLookup("targetLineColor", config, dataset, DEFAULT_CONFIG.targetLineColor),
    targetLineWidth: kvLookup("targetLineWidth", config, dataset, DEFAULT_CONFIG.targetLineWidth),
    // Number formatting
    humanFriendly: kvLookup("humanFriendly", config, dataset, DEFAULT_CONFIG.humanFriendly),
    humanFriendlyDecimal: kvLookup(
      "humanFriendlyDecimal",
      config,
      dataset,
      DEFAULT_CONFIG.humanFriendlyDecimal
    ),
    formatNumber: kvLookup("formatNumber", config, dataset, DEFAULT_CONFIG.formatNumber),
    displayRemaining: kvLookup(
      "displayRemaining",
      config,
      dataset,
      DEFAULT_CONFIG.displayRemaining
    ),
    // Label
    label: kvLookup("label", config, dataset, DEFAULT_CONFIG.label),
    // Title configuration
    title: kvLookup("title", config, dataset, DEFAULT_CONFIG.title),
    titleFontColor: kvLookup("titleFontColor", config, dataset, DEFAULT_CONFIG.titleFontColor),
    titleFontFamily: kvLookup("titleFontFamily", config, dataset, DEFAULT_CONFIG.titleFontFamily),
    titleFontWeight: kvLookup("titleFontWeight", config, dataset, DEFAULT_CONFIG.titleFontWeight),
    titlePosition: kvLookup("titlePosition", config, dataset, DEFAULT_CONFIG.titlePosition),
    // Value font configuration
    valueFontWeight: kvLookup("valueFontWeight", config, dataset, DEFAULT_CONFIG.valueFontWeight),
    // Label font configuration
    labelFontWeight: kvLookup("labelFontWeight", config, dataset, DEFAULT_CONFIG.labelFontWeight),
    // Callbacks
    textRenderer: kvLookup("textRenderer", config, dataset, DEFAULT_CONFIG.textRenderer),
    onAnimationEnd: kvLookup("onAnimationEnd", config, dataset, DEFAULT_CONFIG.onAnimationEnd),
    // Sector colors visualization
    showSectorColors: kvLookup(
      "showSectorColors",
      config,
      dataset,
      DEFAULT_CONFIG.showSectorColors
    )
  };
  return validateConfig(processedConfig);
}
__name(createConfig, "createConfig");
function validateConfig(config) {
  if (!config.id && !config.parentNode) {
    throw new Error("JustGage: Either id or parentNode must be provided");
  }
  if (config.min >= config.max) {
    throw new Error("JustGage: min value must be less than max value");
  }
  if (!Array.isArray(config.levelColors) || config.levelColors.length === 0) {
    config.levelColors = DEFAULT_CONFIG.levelColors;
  }
  return config;
}
__name(validateConfig, "validateConfig");

// src/rendering/svg.js
var createSVGElement = /* @__PURE__ */ __name((tagName) => {
  return document.createElementNS("http://www.w3.org/2000/svg", tagName);
}, "createSVGElement");
var _SVGRenderer = class _SVGRenderer {
  /**
   * Create a new SVG renderer instance
   *
   * @param {HTMLElement} container - DOM element to render SVG into
   * @param {number|string} width - SVG canvas width (pixels or percentage)
   * @param {number|string} height - SVG canvas height (pixels or percentage)
   * @param {number} [viewBoxWidth] - ViewBox width (defaults to width)
   * @param {number} [viewBoxHeight] - ViewBox height (defaults to height)
   */
  constructor(container, width, height, viewBoxWidth, viewBoxHeight) {
    this.container = container;
    this.width = width;
    this.height = height;
    this.viewBoxWidth = viewBoxWidth || width;
    this.viewBoxHeight = viewBoxHeight || height;
    this.svg = null;
    this.elements = /* @__PURE__ */ new Map();
    this.init();
  }
  init() {
    this.svg = createSVGElement("svg");
    this.svg.setAttribute("width", this.width);
    this.svg.setAttribute("height", this.height);
    this.svg.setAttribute("viewBox", `0 0 ${this.viewBoxWidth} ${this.viewBoxHeight}`);
    this.svg.style.overflow = "hidden";
    if (typeof this.width === "string" && this.width.includes("%")) {
      this.svg.setAttribute("preserveAspectRatio", "xMidYMid meet");
    }
    this.container.innerHTML = "";
    this.container.appendChild(this.svg);
  }
  /**
   * Create a circle element
   */
  circle(cx, cy, radius) {
    const circle = createSVGElement("circle");
    circle.setAttribute("cx", cx);
    circle.setAttribute("cy", cy);
    circle.setAttribute("r", radius);
    this.svg.appendChild(circle);
    return new SVGElement(circle);
  }
  /**
   * Create a rectangle element
   */
  rect(x, y, width, height) {
    const rect = createSVGElement("rect");
    rect.setAttribute("x", x);
    rect.setAttribute("y", y);
    rect.setAttribute("width", width);
    rect.setAttribute("height", height);
    this.svg.appendChild(rect);
    return new SVGElement(rect);
  }
  /**
   * Create a path element
   */
  path(pathData) {
    const path = createSVGElement("path");
    path.setAttribute("d", pathData);
    this.svg.appendChild(path);
    return new SVGElement(path);
  }
  /**
   * Create a line element
   */
  line(x1, y1, x2, y2) {
    const line = createSVGElement("line");
    line.setAttribute("x1", x1);
    line.setAttribute("y1", y1);
    line.setAttribute("x2", x2);
    line.setAttribute("y2", y2);
    this.svg.appendChild(line);
    return new SVGElement(line);
  }
  /**
   * Create a text element
   */
  text(x, y, content) {
    const text = createSVGElement("text");
    text.setAttribute("x", x);
    text.setAttribute("y", y);
    text.textContent = content;
    this.svg.appendChild(text);
    return new SVGElement(text);
  }
  /**
   * Create an arc/sector path
   */
  sector(cx, cy, r1, r2, startAngle, endAngle) {
    const pathData = this.createSectorPath(cx, cy, r1, r2, startAngle, endAngle);
    return this.path(pathData);
  }
  /**
   * Generate SVG path data for an arc sector matching original JustGage
   */
  createSectorPath(cx, cy, r1, r2, startAngle, endAngle) {
    const rad1 = (startAngle - 90) * Math.PI / 180;
    const rad2 = (endAngle - 90) * Math.PI / 180;
    const x1 = cx + r1 * Math.cos(rad1);
    const y1 = cy + r1 * Math.sin(rad1);
    const x2 = cx + r2 * Math.cos(rad1);
    const y2 = cy + r2 * Math.sin(rad1);
    const x3 = cx + r2 * Math.cos(rad2);
    const y3 = cy + r2 * Math.sin(rad2);
    const x4 = cx + r1 * Math.cos(rad2);
    const y4 = cy + r1 * Math.sin(rad2);
    let angleSpan = endAngle - startAngle;
    if (angleSpan <= 0) {
      angleSpan += 360;
    }
    const largeArcFlag = angleSpan > 180 ? 1 : 0;
    return [
      `M ${x1} ${y1}`,
      `L ${x2} ${y2}`,
      `A ${r2} ${r2} 0 ${largeArcFlag} 1 ${x3} ${y3}`,
      `L ${x4} ${y4}`,
      `A ${r1} ${r1} 0 ${largeArcFlag} 0 ${x1} ${y1}`,
      "Z"
    ].join(" ");
  }
  /**
   * Create gauge path
   * @param {number|{from: number, to: number}} sectorPctOrValue
   * @param {number} min
   * @param {number} max
   * @param {number} widgetW
   * @param {number} widgetH
   * @param {number} dx
   * @param {number} dy
   * @param {number} gaugeWidthScale
   * @param {boolean} donut
   * @param {boolean} isDiff
   * @returns {string} SVG path data for the gauge
   */
  createGaugePath(sectorPctOrValue, min, max, widgetW, widgetH, dx, dy, gaugeWidthScale, donut = false, isDiff = false) {
    let alpha;
    let Ro;
    let Ri;
    let Cx;
    let Cy;
    let Xo, Yo, Xi, Yi;
    let Xstart, Ystart, XstartInner, YstartInner;
    let path;
    const sectorWasNumber = typeof sectorPctOrValue === "number";
    if (min < 0 && !isDiff) {
      max -= min;
      if (sectorWasNumber) {
        sectorPctOrValue -= min;
      }
      min = 0;
    }
    if (sectorWasNumber) {
      sectorPctOrValue = { from: min, to: sectorPctOrValue };
    }
    const range = max - min;
    let pctStart, pctEnd;
    if (sectorWasNumber) {
      const deltaVStart = sectorPctOrValue.from - min;
      const deltaVEnd = sectorPctOrValue.to - min;
      pctStart = deltaVStart / range;
      pctEnd = deltaVEnd / range;
    } else if (typeof sectorPctOrValue.from === "number" && typeof sectorPctOrValue.to === "number") {
      if (sectorPctOrValue.from <= 1 && sectorPctOrValue.to <= 1) {
        pctStart = sectorPctOrValue.from;
        pctEnd = sectorPctOrValue.to;
      } else {
        pctStart = sectorPctOrValue.from / 100;
        pctEnd = sectorPctOrValue.to / 100;
      }
    } else {
      const deltaVStart = sectorPctOrValue.from - min;
      const deltaVEnd = sectorPctOrValue.to - min;
      pctStart = deltaVStart / range;
      pctEnd = deltaVEnd / range;
    }
    if (donut) {
      alpha = (1 - 2 * pctEnd) * Math.PI;
      Ro = widgetW / 2 - widgetW / 30;
      Ri = Ro - widgetW / GAUGE_WIDTH_DIVISOR * gaugeWidthScale;
      Cx = widgetW / 2 + dx;
      Cy = widgetH / 2 + dy;
      Xo = Cx + Ro * Math.cos(alpha);
      Yo = Cy - Ro * Math.sin(alpha);
      Xi = Cx + Ri * Math.cos(alpha);
      Yi = Cy - Ri * Math.sin(alpha);
      const alphaStart = (1 - 2 * pctStart) * Math.PI;
      Xstart = Cx + Ro * Math.cos(alphaStart);
      Ystart = Cy - Ro * Math.sin(alphaStart);
      XstartInner = Cx + Ri * Math.cos(alphaStart);
      YstartInner = Cy - Ri * Math.sin(alphaStart);
      const angularSpan = Math.abs(pctEnd - pctStart);
      if (angularSpan >= 0.999) {
        const XmidOuter = Cx + Ro;
        const YmidOuter = Cy;
        const XmidInner = Cx + Ri;
        const YmidInner = Cy;
        path = "M" + XstartInner + "," + YstartInner + " ";
        path += "L" + Xstart + "," + Ystart + " ";
        path += "A" + Ro + "," + Ro + " 0 0 1 " + XmidOuter + "," + YmidOuter + " ";
        path += "A" + Ro + "," + Ro + " 0 0 1 " + Xstart + "," + Ystart + " ";
        path += "L" + XstartInner + "," + YstartInner + " ";
        path += "A" + Ri + "," + Ri + " 0 0 0 " + XmidInner + "," + YmidInner + " ";
        path += "A" + Ri + "," + Ri + " 0 0 0 " + XstartInner + "," + YstartInner + " ";
        path += "Z ";
      } else {
        const largeArcFlag = angularSpan > 0.5 ? 1 : 0;
        path = "M" + XstartInner + "," + YstartInner + " ";
        path += "L" + Xstart + "," + Ystart + " ";
        path += "A" + Ro + "," + Ro + " 0 " + largeArcFlag + " 1 " + Xo + "," + Yo + " ";
        path += "L" + Xi + "," + Yi + " ";
        path += "A" + Ri + "," + Ri + " 0 " + largeArcFlag + " 0 " + XstartInner + "," + YstartInner + " ";
        path += "Z ";
      }
    } else if (isDiff) {
      alpha = (1 - pctEnd) * Math.PI;
      Ro = widgetW / 2 - widgetW / 10;
      Ri = Ro - widgetW / GAUGE_WIDTH_DIVISOR * gaugeWidthScale;
      Cx = widgetW / 2 + dx;
      Cy = widgetH / 1.25 + dy;
      Xo = Cx + Ro * Math.cos(alpha);
      Yo = Cy - Ro * Math.sin(alpha);
      Xi = Cx + Ri * Math.cos(alpha);
      Yi = Cy - Ri * Math.sin(alpha);
      const middlePct = 0.5;
      const So = pctEnd < middlePct ? 1 : 0;
      const Si = pctEnd < middlePct ? 0 : 1;
      path = "M" + Cx + "," + (Cy - Ri) + " ";
      path += "L" + Cx + "," + (Cy - Ro) + " ";
      path += "A" + Ro + "," + Ro + " 0 0 " + Si + " " + Xo + "," + Yo + " ";
      path += "L" + Xi + "," + Yi + " ";
      path += "A" + Ri + "," + Ri + " 0 0 " + So + " " + Cx + "," + (Cy - Ri) + " ";
      path += "Z ";
    } else {
      alpha = (1 - pctEnd) * Math.PI;
      Ro = widgetW / 2 - widgetW / 10;
      Ri = Ro - widgetW / GAUGE_WIDTH_DIVISOR * gaugeWidthScale;
      Cx = widgetW / 2 + dx;
      Cy = widgetH / 1.25 + dy;
      Xo = Cx + Ro * Math.cos(alpha);
      Yo = Cy - Ro * Math.sin(alpha);
      Xi = Cx + Ri * Math.cos(alpha);
      Yi = Cy - Ri * Math.sin(alpha);
      const alphaStart = (1 - pctStart) * Math.PI;
      Xstart = Cx + Ro * Math.cos(alphaStart);
      Ystart = Cy - Ro * Math.sin(alphaStart);
      XstartInner = Cx + Ri * Math.cos(alphaStart);
      YstartInner = Cy - Ri * Math.sin(alphaStart);
      path = "M" + XstartInner + "," + YstartInner + " ";
      path += "L" + Xstart + "," + Ystart + " ";
      path += "A" + Ro + "," + Ro + " 0 0 1 " + Xo + "," + Yo + " ";
      path += "L" + Xi + "," + Yi + " ";
      path += "A" + Ri + "," + Ri + " 0 0 0 " + XstartInner + "," + YstartInner + " ";
      path += "Z ";
    }
    return path;
  }
  /**
   * Create gauge pointer using path data (removed - now using direct path creation)
   * Pointers are now created directly using the path() method with original JustGage algorithm
   */
  /**
   * Remove all elements from SVG
   */
  clear() {
    while (this.svg.firstChild) {
      this.svg.removeChild(this.svg.firstChild);
    }
    this.elements.clear();
  }
  /**
   * Remove the entire SVG from DOM
   */
  remove() {
    if (this.svg && this.svg.parentNode) {
      this.svg.parentNode.removeChild(this.svg);
    }
    this.elements.clear();
  }
  /**
   * Create or get defs element for filters and gradients
   * @returns {SVGElement} The defs element
   */
  getDefs() {
    let defs = this.svg.querySelector("defs");
    if (!defs) {
      defs = createSVGElement("defs");
      this.svg.appendChild(defs);
    }
    return defs;
  }
  /**
   * Generate shadow filter for inner shadow effect
   * @param {string} shadowId - Unique ID for the shadow filter
   * @param {object} shadowConfig - Shadow configuration options
   * @param {number} shadowConfig.verticalOffset - Vertical offset for shadow
   * @param {number} shadowConfig.size - Blur size for shadow
   * @param {number} shadowConfig.opacity - Shadow opacity (0-1)
   * @returns {string} The shadow filter ID
   */
  createShadowFilter(shadowId, shadowConfig) {
    const defs = this.getDefs();
    const existingFilter = defs.querySelector(`#${shadowId}`);
    if (existingFilter) {
      existingFilter.remove();
    }
    const filter = createSVGElement("filter");
    filter.setAttribute("id", shadowId);
    defs.appendChild(filter);
    const feOffset = createSVGElement("feOffset");
    feOffset.setAttribute("dx", 0);
    feOffset.setAttribute("dy", shadowConfig.verticalOffset || 0);
    filter.appendChild(feOffset);
    const feGaussianBlur = createSVGElement("feGaussianBlur");
    feGaussianBlur.setAttribute("result", "offset-blur");
    feGaussianBlur.setAttribute("stdDeviation", shadowConfig.size || 0);
    filter.appendChild(feGaussianBlur);
    const feComposite1 = createSVGElement("feComposite");
    feComposite1.setAttribute("operator", "out");
    feComposite1.setAttribute("in", "SourceGraphic");
    feComposite1.setAttribute("in2", "offset-blur");
    feComposite1.setAttribute("result", "inverse");
    filter.appendChild(feComposite1);
    const feFlood = createSVGElement("feFlood");
    feFlood.setAttribute("flood-color", "black");
    feFlood.setAttribute("flood-opacity", shadowConfig.opacity || 0.5);
    feFlood.setAttribute("result", "color");
    filter.appendChild(feFlood);
    const feComposite2 = createSVGElement("feComposite");
    feComposite2.setAttribute("operator", "in");
    feComposite2.setAttribute("in", "color");
    feComposite2.setAttribute("in2", "inverse");
    feComposite2.setAttribute("result", "shadow");
    filter.appendChild(feComposite2);
    const feComposite3 = createSVGElement("feComposite");
    feComposite3.setAttribute("operator", "over");
    feComposite3.setAttribute("in", "shadow");
    feComposite3.setAttribute("in2", "SourceGraphic");
    filter.appendChild(feComposite3);
    return shadowId;
  }
  /**
   * Apply shadow filter to elements
   * @param {string} shadowId - Shadow filter ID
   * @param {SVGElement[]} elements - Elements to apply shadow to
   */
  applyShadowToElements(shadowId, elements) {
    elements.forEach((element) => {
      if (element && element.attr) {
        element.attr({ filter: `url(#${shadowId})` });
      }
    });
  }
  /**
   * Remove shadow filter from elements
   * @param {SVGElement[]} elements - Elements to remove shadow from
   */
  removeShadowFromElements(elements) {
    elements.forEach((element) => {
      if (element && element.attr) {
        element.attr({ filter: "none" });
      }
    });
  }
};
__name(_SVGRenderer, "SVGRenderer");
var SVGRenderer = _SVGRenderer;
var _SVGElement = class _SVGElement {
  constructor(element) {
    this.element = element;
  }
  /**
   * Set element attributes
   */
  attr(attrs) {
    if (typeof attrs === "string") {
      return this.element.getAttribute(attrs);
    }
    Object.keys(attrs).forEach((key) => {
      const value = attrs[key];
      switch (key) {
        case "text":
          this.element.textContent = value;
          break;
        case "fill":
          this.element.setAttribute("fill", value);
          break;
        case "stroke":
          this.element.setAttribute("stroke", value);
          break;
        case "stroke-width":
        case "strokeWidth":
          this.element.setAttribute("stroke-width", value);
          break;
        case "opacity":
          this.element.setAttribute("opacity", value);
          break;
        case "font-family":
        case "fontFamily":
          this.element.setAttribute("font-family", value);
          break;
        case "font-size":
        case "fontSize":
          this.element.setAttribute("font-size", value);
          break;
        case "font-weight":
        case "fontWeight":
          this.element.setAttribute("font-weight", value);
          break;
        case "text-anchor":
        case "textAnchor":
          this.element.setAttribute("text-anchor", value);
          break;
        case "dominant-baseline":
        case "dominantBaseline":
          this.element.setAttribute("dominant-baseline", value);
          break;
        case "filter":
          this.element.setAttribute("filter", value);
          break;
        default:
          this.element.setAttribute(key, value);
      }
    });
    return this;
  }
  /**
   * Remove element from DOM
   */
  remove() {
    if (this.element && this.element.parentNode) {
      this.element.parentNode.removeChild(this.element);
    }
    return this;
  }
  /**
   * Hide element
   */
  hide() {
    this.element.style.display = "none";
    return this;
  }
  /**
   * Show element
   */
  show() {
    this.element.style.display = "";
    return this;
  }
  /**
   * Set element text content
   */
  text(content) {
    if (content === void 0) {
      return this.element.textContent;
    }
    this.element.textContent = content;
    return this;
  }
  /**
   * Apply transform to element
   * @param {string} transform - Transform string (e.g., 'rotate(90 50 50)')
   */
  transform(transform) {
    if (transform === void 0) {
      return this.element.getAttribute("transform");
    }
    this.element.setAttribute("transform", transform);
    return this;
  }
};
__name(_SVGElement, "SVGElement");
var SVGElement = _SVGElement;

// src/utils/colors.js
function cutHex(str) {
  return str.charAt(0) === "#" ? str.substring(1, 7) : str;
}
__name(cutHex, "cutHex");
function isHexColor(val) {
  const regExp = /^#([0-9A-Fa-f]{3}){1,2}$/;
  return typeof val === "string" && regExp.test(val);
}
__name(isHexColor, "isHexColor");
function getColor(val, pct, col, noGradient, custSec) {
  let percentage, rval, gval, bval, lower, upper, range, rangePct, pctLower, pctUpper, color;
  const cust = custSec && custSec.ranges && custSec.ranges.length > 0;
  noGradient = noGradient || cust;
  if (cust) {
    if (custSec.percents === true) val = pct * 100;
    for (let i = 0; i < custSec.ranges.length; i++) {
      if (val >= custSec.ranges[i].lo && val <= custSec.ranges[i].hi) {
        return custSec.ranges[i].color;
      }
    }
  }
  const no = col.length;
  if (no === 1) return col[0];
  const inc = noGradient ? 1 / no : 1 / (no - 1);
  const colors = [];
  for (let i = 0; i < col.length; i++) {
    percentage = noGradient ? inc * (i + 1) : inc * i;
    rval = parseInt(cutHex(col[i]).substring(0, 2), 16);
    gval = parseInt(cutHex(col[i]).substring(2, 4), 16);
    bval = parseInt(cutHex(col[i]).substring(4, 6), 16);
    colors[i] = {
      pct: percentage,
      color: {
        r: rval,
        g: gval,
        b: bval
      }
    };
  }
  if (pct === 0) {
    return `rgb(${[colors[0].color.r, colors[0].color.g, colors[0].color.b].join(",")})`;
  }
  for (let j = 0; j < colors.length; j++) {
    if (pct <= colors[j].pct) {
      if (noGradient) {
        return `rgb(${[colors[j].color.r, colors[j].color.g, colors[j].color.b].join(",")})`;
      } else {
        lower = colors[j - 1] || colors[0];
        upper = colors[j];
        range = upper.pct - lower.pct;
        rangePct = (pct - lower.pct) / range;
        pctLower = 1 - rangePct;
        pctUpper = rangePct;
        color = {
          r: Math.floor(lower.color.r * pctLower + upper.color.r * pctUpper),
          g: Math.floor(lower.color.g * pctLower + upper.color.g * pctUpper),
          b: Math.floor(lower.color.b * pctLower + upper.color.b * pctUpper)
        };
        return `rgb(${[color.r, color.g, color.b].join(",")})`;
      }
    }
  }
}
__name(getColor, "getColor");

// src/utils/formatters.js
function humanFriendlyNumber(n, d) {
  const d2 = Math.pow(10, d);
  const s = " KMGTPE";
  let i = 0;
  const c = 1e3;
  while ((n >= c || n <= -c) && ++i < s.length) {
    n = n / c;
  }
  i = i >= s.length ? s.length - 1 : i;
  return Math.round(n * d2) / d2 + s[i];
}
__name(humanFriendlyNumber, "humanFriendlyNumber");
function formatNumber(x) {
  const parts = x.toString().split(".");
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return parts.join(".");
}
__name(formatNumber, "formatNumber");

// src/core/GaugeAnimator.js
var _GaugeAnimator = class _GaugeAnimator {
  constructor() {
    this.currentAnimation = null;
  }
  /**
   * Animate gauge from one value to another
   * @param {Object} options - Animation options
   * @param {number} options.fromValue - Starting value
   * @param {number} options.toValue - Target value
   * @param {number} options.duration - Animation duration in ms
   * @param {string} options.easing - Easing type ('linear', '>', '<', '<>', 'bounce')
   * @param {Function} options.onUpdate - Called each frame with current value
   * @param {Function} [options.onComplete] - Called when animation completes
   * @param {Function} [options.onCounterUpdate] - Called for counter text updates
   */
  animate({
    fromValue,
    toValue,
    duration,
    easing = "linear",
    onUpdate,
    onComplete,
    onCounterUpdate
  }) {
    this.cancel();
    if (duration <= 0) {
      if (onUpdate) onUpdate(toValue);
      if (onCounterUpdate) onCounterUpdate(toValue);
      if (onComplete) onComplete();
      return;
    }
    const startTime = Date.now();
    const valueRange = toValue - fromValue;
    const animate = /* @__PURE__ */ __name(() => {
      const elapsed = Date.now() - startTime;
      const progress = Math.min(elapsed / duration, 1);
      const easedProgress = this._applyEasing(progress, easing);
      const currentValue = fromValue + valueRange * easedProgress;
      if (onUpdate) onUpdate(currentValue);
      if (onCounterUpdate) onCounterUpdate(currentValue);
      if (progress < 1) {
        this.currentAnimation = requestAnimationFrame(animate);
      } else {
        this.currentAnimation = null;
        if (onUpdate) onUpdate(toValue);
        if (onCounterUpdate) onCounterUpdate(toValue);
        if (onComplete) onComplete();
      }
    }, "animate");
    this.currentAnimation = requestAnimationFrame(animate);
  }
  /**
   * Cancel current animation
   */
  cancel() {
    if (this.currentAnimation) {
      cancelAnimationFrame(this.currentAnimation);
      this.currentAnimation = null;
    }
  }
  /**
   * Apply easing function to progress
   * Ref: https://github.com/DmitryBaranovskiy/raphael/blob/master/raphael.js#L4161
   * @private
   */
  _applyEasing(progress, easing) {
    switch (easing) {
      case "linear":
      case "-":
        return progress;
      case ">":
      case "easeOut":
      case "ease-out":
        return Math.pow(progress, 0.48);
      case "<":
      case "easeIn":
      case "ease-in":
        return Math.pow(progress, 1.7);
      case "<>":
      case "easeInOut":
      case "ease-in-out":
        return progress < 0.5 ? 2 * progress * progress : 1 - Math.pow(-2 * progress + 2, 2) / 2;
      case "bounce":
        return this._bounceEasing(progress);
      case "elastic":
        return Math.pow(2, -10 * progress) * Math.sin((progress - 0.075) * (2 * Math.PI) / 0.3) + 1;
      case "backIn":
      case "back-in": {
        const c1 = 1.70158;
        const c3 = c1 + 1;
        return c3 * progress * progress * progress - c1 * progress * progress;
      }
      case "backOut":
      case "back-out": {
        const c2 = 1.70158;
        const c4 = c2 + 1;
        return 1 + c4 * Math.pow(progress - 1, 3) + c2 * Math.pow(progress - 1, 2);
      }
      default:
        return progress;
    }
  }
  /**
   * Bounce easing function
   * @private
   */
  _bounceEasing(t) {
    const n1 = 7.5625;
    const d1 = 2.75;
    if (t < 1 / d1) {
      return n1 * t * t;
    } else if (t < 2 / d1) {
      return n1 * (t -= 1.5 / d1) * t + 0.75;
    } else if (t < 2.5 / d1) {
      return n1 * (t -= 2.25 / d1) * t + 0.9375;
    } else {
      return n1 * (t -= 2.625 / d1) * t + 0.984375;
    }
  }
  /**
   * Check if animation is currently running
   */
  isAnimating() {
    return this.currentAnimation !== null;
  }
};
__name(_GaugeAnimator, "GaugeAnimator");
var GaugeAnimator = _GaugeAnimator;

// src/core/JustGage.js
var _JustGage = class _JustGage {
  /**
   * Create a new gauge instance
   *
   * @param {import('../types/index.d.ts').JustGageConfig} config - Configuration options for the gauge
   * @throws {Error} When no configuration object is provided
   * @throws {Error} When neither id nor parentNode is provided
   * @throws {Error} When specified DOM element is not found
   * @throws {Error} When min >= max
   */
  constructor(config) {
    if (!config) {
      throw new Error("JustGage: Configuration object is required");
    }
    if (config.id) {
      this.node = document.getElementById(config.id);
      if (!this.node) {
        throw new Error(`JustGage: No element with id '${config.id}' found`);
      }
    } else if (config.parentNode) {
      this.node = config.parentNode;
    } else {
      throw new Error("JustGage: Either id or parentNode must be provided");
    }
    const dataset = this.node.dataset || {};
    this.config = createConfig(config, dataset);
    this.originalValue = config.value ?? -1;
    this._initializeGauge();
  }
  /**
   * Initialize the gauge rendering
   * @private
   */
  _initializeGauge() {
    let width, height, viewBoxWidth, viewBoxHeight;
    if (this.config.relativeGaugeSize) {
      width = "100%";
      height = "100%";
      if (this.config.donut) {
        viewBoxWidth = 200;
        viewBoxHeight = this.config.title && this.config.title.length > 0 ? 240 : 200;
      } else {
        viewBoxWidth = 200;
        viewBoxHeight = this.config.title && this.config.title.length > 0 ? 150 : 100;
      }
    } else {
      width = this.config.width;
      height = this.config.height;
      if (!width || !height) {
        const rect = this.node.getBoundingClientRect();
        if (!width) width = rect.width || 200;
        if (!height) height = rect.height || 100;
        this.config.width = width;
        this.config.height = height;
      }
      viewBoxWidth = width;
      viewBoxHeight = height;
    }
    this.renderer = new SVGRenderer(this.node, width, height, viewBoxWidth, viewBoxHeight);
    this.canvas = {
      gauge: null,
      level: null,
      title: null,
      value: null,
      min: null,
      max: null,
      pointer: null
    };
    this.animator = new GaugeAnimator();
    this._drawGauge();
    if (this.config.showInnerShadow) {
      this._initializeShadow();
    }
    this._startInitialAnimation();
  }
  /**
   * Draw the complete gauge
   * @private
   */
  _drawGauge() {
    const config = this.config;
    const { widgetW, widgetH, dx, dy } = this._calculateGaugeGeometry();
    const gaugePath = this.renderer.createGaugePath(
      config.max,
      config.min,
      config.max,
      widgetW,
      widgetH,
      dx,
      dy,
      config.gaugeWidthScale || 1,
      config.donut,
      false
      // this is relevant only for level drawing
    );
    this.canvas.gauge = this.renderer.path(gaugePath).attr({
      fill: config.gaugeColor,
      stroke: "none"
    });
    this._applyDonutRotation(this.canvas.gauge, config, widgetW, widgetH, dx, dy);
    if (config.showSectorColors) {
      this._drawSectorColors();
    } else {
      const startValue = config.differential ? (config.max + config.min) / 2 : config.min;
      this._drawLevel(startValue);
    }
    this._drawLabels();
    if (config.pointer) {
      this._drawPointer();
    }
    this._drawTargetLine();
  }
  /**
   * Draw gauge level (filled arc)
   * @private
   * @param {number} [animateValue] - If provided, draws level at this value instead of config.value
   */
  _drawLevel(animateValue) {
    const config = this.config;
    if (config.showSectorColors) {
      return;
    }
    const targetValue = animateValue !== void 0 ? animateValue : config.value;
    const clampedValue = this._clampValue(targetValue);
    const { widgetW, widgetH, dx, dy } = this._calculateGaugeGeometry();
    let displayValue = clampedValue;
    if (config.reverse) {
      displayValue = config.max + config.min - clampedValue;
    }
    const color = this._getLevelColor(clampedValue);
    const levelPath = this.renderer.createGaugePath(
      displayValue,
      config.min,
      config.max,
      widgetW,
      widgetH,
      dx,
      dy,
      config.gaugeWidthScale || 1,
      config.donut,
      config.differential
    );
    if (this.canvas.level) {
      this.canvas.level.attr({
        d: levelPath,
        fill: color
      });
    } else {
      this.canvas.level = this.renderer.path(levelPath).attr({
        fill: color,
        stroke: "none"
      });
      this._applyDonutRotation(this.canvas.level, config, widgetW, widgetH, dx, dy);
    }
  }
  /**
   * Draw sector colors as filled arcs
   * @private
   */
  _drawSectorColors() {
    const config = this.config;
    const { widgetW, widgetH, dx, dy } = this._calculateGaugeGeometry();
    if (this.canvas.sectors) {
      this.canvas.sectors.forEach((sector) => sector.remove());
    }
    this.canvas.sectors = [];
    let sectors = [];
    if (config.customSectors && config.customSectors.ranges && config.customSectors.ranges.length > 0) {
      sectors = config.customSectors.ranges.map((range) => {
        if (config.customSectors.percents) {
          return {
            lo: range.lo,
            hi: range.hi,
            color: range.color
          };
        } else {
          const min = config.min;
          const max = config.max;
          const span = max - min;
          return {
            lo: (range.lo - min) / span,
            hi: (range.hi - min) / span,
            color: range.color
          };
        }
      });
    } else if (Array.isArray(config.levelColors) && config.levelColors.length > 0) {
      const no = config.levelColors.length;
      const inc = 1 / no;
      sectors = config.levelColors.map((color, i) => {
        const startPct = i * inc;
        const endPct = (i + 1) * inc;
        return {
          lo: config.min + (config.max - config.min) * startPct,
          hi: config.min + (config.max - config.min) * endPct,
          color
        };
      });
    } else {
      return;
    }
    for (let i = sectors.length - 1; i >= 0; i--) {
      const sector = sectors[i];
      let sectorMin = sector.lo;
      let sectorMax = sector.hi;
      if (config.reverse) {
        const temp = config.max + config.min - sectorMax;
        sectorMax = config.max + config.min - sectorMin;
        sectorMin = temp;
      }
      const sectorPath = this.renderer.createGaugePath(
        { from: sectorMin, to: sectorMax },
        config.min,
        config.max,
        widgetW,
        widgetH,
        dx,
        dy,
        config.gaugeWidthScale || 1,
        config.donut
      );
      const sectorElement = this.renderer.path(sectorPath).attr({
        fill: sector.color,
        stroke: "none"
      });
      this._applyDonutRotation(sectorElement, config, widgetW, widgetH, dx, dy);
      this.canvas.sectors.push(sectorElement);
    }
  }
  /**
   * Calculate consistent gauge geometry for both arc and text positioning
   * Uses caching to avoid redundant calculations
   * @private
   */
  _calculateGaugeGeometry() {
    const config = this.config;
    let w, h;
    if (config.relativeGaugeSize) {
      w = this.renderer.viewBoxWidth;
      h = this.renderer.viewBoxHeight;
    } else {
      w = config.width;
      h = config.height;
    }
    let widgetW, widgetH, dx, dy;
    if (config.donut) {
      const size = Math.min(w, h);
      widgetW = size;
      widgetH = size;
      dx = (w - widgetW) / 2;
      dy = (h - widgetH) / 2;
    } else {
      if (w > h) {
        widgetH = h;
        widgetW = widgetH * 2;
        if (config.title.length > 0) {
          widgetW = widgetH * 1.25;
        }
        if (widgetW > w) {
          const aspect = widgetW / w;
          widgetW = widgetW / aspect;
          widgetH = widgetH / aspect;
        }
      } else if (w < h) {
        widgetW = w;
        widgetH = widgetW / 2;
        if (config.title.length > 0) {
          widgetH = widgetW / 1.25;
        }
      } else {
        widgetW = w;
        widgetH = widgetW * 0.5;
        if (config.title.length > 0) {
          widgetH = widgetW * 0.75;
        }
      }
      dx = (w - widgetW) / 2;
      dy = (h - widgetH) / 2;
      if (config.titlePosition === "below") {
        dy -= widgetH / 6.4;
      }
    }
    const cx = dx + widgetW / 2;
    const cy = config.donut ? dy + widgetH / 2 : dy + widgetH / 1.25;
    const outerRadius = config.donut ? widgetW / 2 - widgetW / 30 : widgetW / 2 - widgetW / 10;
    const gaugeWidthScale = config.gaugeWidthScale || 1;
    const innerRadius = outerRadius - widgetW / GAUGE_WIDTH_DIVISOR * gaugeWidthScale;
    return { cx, cy, outerRadius, innerRadius, widgetW, widgetH, dx, dy };
  }
  /**
   * Calculate font sizes for different text elements
   * @param {number} widgetH - Widget height
   * @param {object} config - Configuration object
   * @returns {object} Object containing all calculated font sizes
   * @private
   */
  _calculateFontSizes(widgetH, config) {
    const titleFontSize = widgetH / 8 > config.titleMinFontSize ? widgetH / 10 : config.titleMinFontSize;
    const valueFontSize = config.donut ? widgetH / 6.4 > 16 ? widgetH / 5.4 : 18 : widgetH / 6.5 > config.valueMinFontSize ? widgetH / 6.5 : config.valueMinFontSize;
    const labelFontSize = config.donut ? widgetH / 16 > 10 ? widgetH / 16 : 10 : widgetH / 16 > config.labelMinFontSize ? widgetH / 16 : config.labelMinFontSize;
    const minMaxLabelFontSize = config.donut ? widgetH / 16 > 10 ? widgetH / 16 : 10 : widgetH / 16 > config.minLabelMinFontSize ? widgetH / 16 : config.minLabelMinFontSize;
    return {
      title: titleFontSize,
      value: valueFontSize,
      label: labelFontSize,
      minMax: minMaxLabelFontSize
    };
  }
  /**
   * Apply donut rotation to an element if donut mode is enabled
   * @param {object} element - SVG element to rotate
   * @param {object} config - Configuration object
   * @param {number} widgetW - Widget width
   * @param {number} widgetH - Widget height
   * @param {number} dx - X offset
   * @param {number} dy - Y offset
   * @private
   */
  _applyDonutRotation(element, config, widgetW, widgetH, dx, dy, rotationOverride = null) {
    if (config.donut) {
      const centerX = widgetW / 2 + dx;
      const centerY = widgetH / 2 + dy;
      const rotation = rotationOverride || config.donutStartAngle || 90;
      element.transform(`rotate(${rotation} ${centerX} ${centerY})`);
    }
  }
  /**
   * Draw text labels
   * @private
   */
  _drawLabels() {
    const config = this.config;
    const { widgetW, widgetH, dx, dy } = this._calculateGaugeGeometry();
    const fontSizes = this._calculateFontSizes(widgetH, config);
    if (config.title) {
      const titleX = dx + widgetW / 2;
      let titleY;
      if (config.donut) {
        titleY = dy + (config.titlePosition === "below" ? widgetH + 15 : -5);
      } else {
        titleY = dy + (config.titlePosition === "below" ? widgetH * 1.07 : widgetH / 6.4);
      }
      this.canvas.title = this.renderer.text(titleX, titleY, config.title).attr({
        "font-family": config.titleFontFamily,
        "font-size": fontSizes.title,
        "font-weight": config.titleFontWeight,
        "text-anchor": "middle",
        fill: config.titleFontColor
      });
    }
    const valueX = dx + widgetW / 2;
    const valueY = config.donut ? config.label ? dy + widgetH / 1.85 : dy + widgetH / 1.7 : dy + widgetH / 1.275;
    if (!config.hideValue) {
      const displayValue = this._formatValue(config.value);
      this.canvas.value = this.renderer.text(valueX, valueY, displayValue).attr({
        "font-family": config.valueFontFamily,
        "font-size": fontSizes.value,
        "font-weight": "bold",
        "text-anchor": "middle",
        fill: config.valueFontColor
      });
    }
    if (config.label) {
      const labelY = config.donut ? valueY + fontSizes.label : valueY + fontSizes.value / 2 + 5;
      this.canvas.label = this.renderer.text(valueX, labelY, config.label).attr({
        "font-family": config.labelFontFamily,
        "font-size": fontSizes.label,
        "text-anchor": "middle",
        fill: config.labelFontColor
      });
    }
    if (config.showMinMax && !config.hideMinMax && !config.donut) {
      const gaugeWidthScale = config.gaugeWidthScale || 1;
      let minMaxLabelY;
      if (config.donut) {
        minMaxLabelY = valueY + fontSizes.label;
      } else {
        minMaxLabelY = valueY + fontSizes.value / 2 + 5;
      }
      const minX = dx + widgetW / 10 + widgetW / GAUGE_WIDTH_DIVISOR * gaugeWidthScale / 2;
      const maxX = dx + widgetW - widgetW / 10 - widgetW / GAUGE_WIDTH_DIVISOR * gaugeWidthScale / 2;
      const minY = minMaxLabelY;
      const maxY = minMaxLabelY;
      const minText = this._formatDisplayText(config.min, config, "min");
      const maxText = this._formatDisplayText(config.max, config, "max");
      if (!config.reverse) {
        this.canvas.min = this.renderer.text(minX, minY, minText).attr({
          "font-family": config.labelFontFamily,
          "font-size": fontSizes.minMax,
          "text-anchor": "middle",
          fill: config.labelFontColor
        });
        this.canvas.max = this.renderer.text(maxX, maxY, maxText).attr({
          "font-family": config.labelFontFamily,
          "font-size": fontSizes.minMax,
          "text-anchor": "middle",
          fill: config.labelFontColor
        });
      } else {
        this.canvas.min = this.renderer.text(maxX, maxY, minText).attr({
          "font-family": config.labelFontFamily,
          "font-size": fontSizes.minMax,
          "text-anchor": "middle",
          fill: config.labelFontColor
        });
        this.canvas.max = this.renderer.text(minX, minY, maxText).attr({
          "font-family": config.labelFontFamily,
          "font-size": fontSizes.minMax,
          "text-anchor": "middle",
          fill: config.labelFontColor
        });
      }
    }
  }
  /**
   * Draw gauge pointer using original JustGage needle algorithm
   * @private
   */
  _drawPointer() {
    const config = this.config;
    const { widgetW, widgetH, dx, dy } = this._calculateGaugeGeometry();
    const clampedValue = this._clampValue(config.value);
    let value = clampedValue;
    if (config.reverse) {
      value = config.max + config.min - clampedValue;
    }
    const min = config.min;
    const max = config.max;
    const gws = config.gaugeWidthScale;
    const donut = config.donut;
    let dlt = widgetW * 3.5 / 100;
    let dlb = widgetW / 15;
    let dw = widgetW / 100;
    if (config.pointerOptions.toplength != null && config.pointerOptions.toplength !== void 0) {
      dlt = config.pointerOptions.toplength;
    }
    if (config.pointerOptions.bottomlength != null && config.pointerOptions.bottomlength !== void 0) {
      dlb = config.pointerOptions.bottomlength;
    }
    if (config.pointerOptions.bottomwidth != null && config.pointerOptions.bottomwidth !== void 0) {
      dw = config.pointerOptions.bottomwidth;
    }
    let alpha, Ro, Ri, Cy, Xo, Yo, Xi, Yi, Xc, Yc, Xz, Yz, Xa, Ya, Xb, Yb, path;
    if (donut) {
      alpha = (1 - 2 * (value - min) / (max - min)) * Math.PI;
      Ro = widgetW / 2 - widgetW / 30;
      Ri = Ro - widgetW / GAUGE_WIDTH_DIVISOR * gws;
      Cy = widgetH / 2 + dy;
      Xo = widgetW / 2 + dx + Ro * Math.cos(alpha);
      Yo = widgetH - (widgetH - Cy) - Ro * Math.sin(alpha);
      Xi = widgetW / 2 + dx + Ri * Math.cos(alpha);
      Yi = widgetH - (widgetH - Cy) - Ri * Math.sin(alpha);
      Xc = Xo + dlt * Math.cos(alpha);
      Yc = Yo - dlt * Math.sin(alpha);
      Xz = Xi - dlb * Math.cos(alpha);
      Yz = Yi + dlb * Math.sin(alpha);
      Xa = Xz + dw * Math.sin(alpha);
      Ya = Yz + dw * Math.cos(alpha);
      Xb = Xz - dw * Math.sin(alpha);
      Yb = Yz - dw * Math.cos(alpha);
      path = `M${Xa},${Ya} L${Xb},${Yb} L${Xc},${Yc} Z`;
    } else {
      alpha = (1 - (value - min) / (max - min)) * Math.PI;
      Ro = widgetW / 2 - widgetW / 10;
      Ri = Ro - widgetW / GAUGE_WIDTH_DIVISOR * gws;
      Cy = widgetH / 1.25 + dy;
      Xo = widgetW / 2 + dx + Ro * Math.cos(alpha);
      Yo = widgetH - (widgetH - Cy) - Ro * Math.sin(alpha);
      Xi = widgetW / 2 + dx + Ri * Math.cos(alpha);
      Yi = widgetH - (widgetH - Cy) - Ri * Math.sin(alpha);
      Xc = Xo + dlt * Math.cos(alpha);
      Yc = Yo - dlt * Math.sin(alpha);
      Xz = Xi - dlb * Math.cos(alpha);
      Yz = Yi + dlb * Math.sin(alpha);
      Xa = Xz + dw * Math.sin(alpha);
      Ya = Yz + dw * Math.cos(alpha);
      Xb = Xz - dw * Math.sin(alpha);
      Yb = Yz - dw * Math.cos(alpha);
      path = `M${Xa},${Ya} L${Xb},${Yb} L${Xc},${Yc} Z`;
    }
    this.canvas.pointer = this.renderer.path(path).attr({
      fill: config.pointerOptions.color || "#000000",
      stroke: config.pointerOptions.stroke || "none",
      "stroke-width": config.pointerOptions.stroke_width || 0,
      "stroke-linecap": config.pointerOptions.stroke_linecap || "square"
    });
    this._applyDonutRotation(
      this.canvas.pointer,
      config,
      widgetW,
      widgetH,
      dx,
      dy,
      config.donutStartAngle || 90
    );
  }
  /**
   * Draw target line at specified value
   * @private
   */
  _drawTargetLine() {
    const config = this.config;
    if (config.targetLine == null) {
      return;
    }
    const { widgetW, widgetH, dx, dy } = this._calculateGaugeGeometry();
    let targetValue = config.targetLine;
    if (config.reverse) {
      targetValue = config.max + config.min - config.targetLine;
    }
    let alpha;
    if (config.donut) {
      alpha = (1 - 2 * (targetValue - config.min) / (config.max - config.min)) * Math.PI;
    } else {
      alpha = (1 - (targetValue - config.min) / (config.max - config.min)) * Math.PI;
    }
    let Ro = widgetW / 2 - widgetW / 10;
    let Ri = Ro - widgetW / GAUGE_WIDTH_DIVISOR * config.gaugeWidthScale;
    let Cx, Cy, Xo, Yo, Xi, Yi;
    if (config.donut) {
      Ro = widgetW / 2 - widgetW / 30;
      Ri = Ro - widgetW / GAUGE_WIDTH_DIVISOR * config.gaugeWidthScale;
      Cx = widgetW / 2 + dx;
      Cy = widgetH / 2 + dy;
      Xo = Cx + Ro * Math.cos(alpha);
      Yo = Cy - Ro * Math.sin(alpha);
      Xi = Cx + Ri * Math.cos(alpha);
      Yi = Cy - Ri * Math.sin(alpha);
    } else {
      Cx = widgetW / 2 + dx;
      Cy = widgetH / 1.25 + dy;
      Xo = Cx + Ro * Math.cos(alpha);
      Yo = Cy - Ro * Math.sin(alpha);
      Xi = Cx + Ri * Math.cos(alpha);
      Yi = Cy - Ri * Math.sin(alpha);
    }
    const pathData = `M ${Xi} ${Yi} L ${Xo} ${Yo}`;
    this.canvas.targetLine = this.renderer.path(pathData).attr({
      stroke: config.targetLineColor,
      "stroke-width": config.targetLineWidth
    });
    this._applyDonutRotation(this.canvas.targetLine, config, widgetW, widgetH, dx, dy);
  }
  /**
   * Get level color based on value
   * @private
   */
  _getLevelColor(value) {
    const config = this.config;
    const range = config.max - config.min;
    const clampedValue = this._clampValue(value);
    const ratio = (clampedValue - config.min) / range;
    return getColor(
      clampedValue,
      ratio,
      config.levelColors,
      config.noGradient,
      config.customSectors
    );
  }
  /**
   * Format text for display based on configuration
   * @param {number} value - The numeric value to format
   * @param {object} config - Configuration object
   * @param {string} textType - Type of text ('min', 'max', 'value')
   * @returns {string} Formatted text
   * @private
   */
  _formatDisplayText(value, config, textType = "value") {
    if (textType === "min" && config.minTxt) {
      return config.minTxt;
    }
    if (textType === "max" && config.maxTxt) {
      return config.maxTxt;
    }
    if (config.humanFriendly) {
      return humanFriendlyNumber(value, config.humanFriendlyDecimal) + (textType === "value" ? config.symbol : "");
    } else if (config.formatNumber) {
      const formatted = formatNumber(
        textType === "value" ? (value * 1).toFixed(config.decimals) : value
      );
      return formatted + (textType === "value" ? config.symbol : "");
    } else if (textType === "value" && config.displayRemaining) {
      return ((config.max - value) * 1).toFixed(config.decimals) + config.symbol;
    } else {
      const formatted = textType === "value" ? (value * 1).toFixed(config.decimals) : value;
      return formatted + (textType === "value" ? config.symbol : "");
    }
  }
  /**
   * Clamp value to min/max range for visual representation
   * @private
   * @param {number} value - Value to clamp
   * @returns {number} Clamped value within min/max boundaries
   */
  _clampValue(value) {
    const config = this.config;
    return Math.max(config.min, Math.min(config.max, value));
  }
  /**
   * Format value for display
   * @private
   */
  _formatValue(value) {
    const config = this.config;
    if (config.textRenderer && typeof config.textRenderer === "function") {
      const renderedValue = config.textRenderer(value);
      if (renderedValue !== false) {
        return renderedValue;
      }
    }
    return this._formatDisplayText(value, config, "value");
  }
  /**
   * Refresh gauge with new values
   * @param {number} val - New value
   * @param {number} [max] - New maximum value
   * @param {number} [min] - New minimum value
   * @param {string} [label] - New label
   */
  refresh(val, max, min, label) {
    if (!isNumber(val)) {
      throw new Error("JustGage: refresh() requires a numeric value");
    }
    const currentValue = this._getCurrentDisplayValue();
    const originalVal = val;
    if (label !== null && label !== void 0) {
      this.config.label = label;
      if (this.canvas.label) {
        this.canvas.label.attr({ text: this.config.label });
      }
    }
    if (isNumber(min)) {
      this.config.min = min;
      if (this.canvas.min) {
        const minText = this._formatDisplayText(this.config.min, this.config, "min");
        this.canvas.min.attr({ text: minText });
      }
    }
    if (isNumber(max)) {
      this.config.max = max;
      if (this.canvas.max) {
        const maxText = this._formatDisplayText(this.config.max, this.config, "max");
        this.canvas.max.attr({ text: maxText });
      }
    }
    val = val * 1;
    let displayVal = originalVal;
    if (this.config.textRenderer && this.config.textRenderer(displayVal) !== false) {
      displayVal = this.config.textRenderer(displayVal);
    } else if (this.config.humanFriendly) {
      displayVal = humanFriendlyNumber(displayVal, this.config.humanFriendlyDecimal) + this.config.symbol;
    } else if (this.config.formatNumber) {
      displayVal = formatNumber((displayVal * 1).toFixed(this.config.decimals)) + this.config.symbol;
    } else if (this.config.displayRemaining) {
      displayVal = ((this.config.max - displayVal) * 1).toFixed(this.config.decimals) + this.config.symbol;
    } else {
      displayVal = (displayVal * 1).toFixed(this.config.decimals) + this.config.symbol;
    }
    this.config.value = val * 1;
    if (!this.config.counter && !this.config.hideValue && this.canvas.value) {
      this.canvas.value.attr({ text: displayVal });
    }
    this.animator.animate({
      fromValue: currentValue,
      toValue: this.config.value,
      duration: this.config.refreshAnimationTime,
      easing: this.config.refreshAnimationType,
      onUpdate: /* @__PURE__ */ __name((newValue) => {
        this._drawLevel(newValue);
        if (this.config.pointer) {
          this._updatePointer(newValue);
        }
      }, "onUpdate"),
      onCounterUpdate: this.config.counter ? (newValue) => {
        this._updateCounterText(newValue);
      } : null,
      onComplete: /* @__PURE__ */ __name(() => {
        if (this.config.onAnimationEnd && typeof this.config.onAnimationEnd === "function") {
          this.config.onAnimationEnd.call(this);
        }
      }, "onComplete")
    });
  }
  /**
   * Update gauge appearance options
   * @param {object|string} options - Options object or option name
   * @param {any} [val] - Option value (if options is string)
   */
  update(options, val) {
    if (typeof options === "string") {
      this._updateProperty(options, val);
    } else if (options && typeof options === "object") {
      for (const [option, value] of Object.entries(options)) {
        this._updateProperty(option, value);
      }
    }
  }
  /**
   * Update a single property
   * @param {string} option - Property name
   * @param {any} val - Property value
   * @private
   */
  _updateProperty(option, val) {
    switch (option) {
      case "valueFontColor":
        if (!isHexColor(val)) {
          console.warn("JustGage: valueFontColor must be a valid hex color");
          return;
        }
        this.config.valueFontColor = val;
        if (this.canvas.value) {
          this.canvas.value.attr({ fill: val });
        }
        break;
      case "labelFontColor":
        if (!isHexColor(val)) {
          console.warn("JustGage: labelFontColor must be a valid hex color");
          return;
        }
        this.config.labelFontColor = val;
        if (this.canvas.min) {
          this.canvas.min.attr({ fill: val });
        }
        if (this.canvas.max) {
          this.canvas.max.attr({ fill: val });
        }
        if (this.canvas.label) {
          this.canvas.label.attr({ fill: val });
        }
        break;
      case "gaugeColor":
        this.config.gaugeColor = val;
        if (this.canvas.background) {
          this.canvas.background.attr({ fill: val });
        }
        break;
      case "levelColors":
        this.config.levelColors = val;
        if (this.canvas.level) {
          this.canvas.level.remove();
          this._drawLevel();
        }
        break;
      case "targetLine":
        this.config.targetLine = val;
        if (this.canvas.targetLine) {
          this.canvas.targetLine.remove();
          this.canvas.targetLine = null;
        }
        if (val !== null && val !== void 0) {
          this._drawTargetLine();
        }
        break;
      case "targetLineColor":
        this.config.targetLineColor = val;
        if (this.canvas.targetLine) {
          this.canvas.targetLine.attr({ stroke: val });
        }
        break;
      case "targetLineWidth":
        this.config.targetLineWidth = val;
        if (this.canvas.targetLine) {
          this.canvas.targetLine.attr({ "stroke-width": val });
        }
        break;
      case "symbol":
        this.config.symbol = val;
        if (this.canvas.value) {
          const displayValue = this._formatValue(this.config.value);
          this.canvas.value.attr({ text: displayValue });
        }
        break;
      case "decimals":
        this.config.decimals = val;
        if (this.canvas.value) {
          const displayValue = this._formatValue(this.config.value);
          this.canvas.value.attr({ text: displayValue });
        }
        break;
      case "title":
        this.config.title = val;
        if (this.canvas.title) {
          this.canvas.title.attr({ text: val || "" });
        } else if (val) {
          const geometry = this._calculateGaugeGeometry();
          const fontSizes = this._calculateFontSizes(geometry.widgetH, this.config);
          const { titleX, titleY } = geometry;
          this.canvas.title = this.renderer.text(titleX, titleY, val).attr({
            "font-family": this.config.titleFontFamily,
            "font-size": fontSizes.title,
            "font-weight": this.config.titleFontWeight,
            "text-anchor": "middle",
            fill: this.config.titleFontColor
          });
        }
        break;
      case "titleFontColor":
        if (!isHexColor(val)) {
          console.warn("JustGage: titleFontColor must be a valid hex color");
          return;
        }
        this.config.titleFontColor = val;
        if (this.canvas.title) {
          this.canvas.title.attr({ fill: val });
        }
        break;
      case "showSectorColors":
        this.config.showSectorColors = val;
        if (this.canvas.sectors) {
          this.canvas.sectors.forEach((sector) => sector.remove());
          this.canvas.sectors = null;
        }
        if (this.canvas.level) {
          this.canvas.level.remove();
          this.canvas.level = null;
        }
        if (val) {
          this._drawSectorColors();
        } else {
          this._drawLevel();
        }
        break;
      case "customSectors":
        this.config.customSectors = val;
        if (this.config.showSectorColors) {
          this._drawSectorColors();
        } else if (this.canvas.level) {
          this.canvas.level.remove();
          this.canvas.level = null;
          this._drawLevel();
        }
        break;
      default:
        console.warn(`JustGage: "${option}" is not a supported update setting`);
    }
  }
  /**
   * Destroy the gauge and clean up resources
   */
  destroy() {
    if (this.animator) {
      this.animator.cancel();
    }
    if (this.renderer) {
      this.renderer.remove();
    }
    if (this.node?.parentNode) {
      this.node.innerHTML = "";
    }
    this.node = null;
    this.config = null;
    this.renderer = null;
    this.animator = null;
    this.canvas = null;
  }
  /**
   * Get current gauge value
   * @returns {number} Current value
   */
  getValue() {
    return this.config.value;
  }
  /**
   * Get current configuration
   * @returns {object} Current configuration
   */
  getConfig() {
    return { ...this.config };
  }
  /**
   * Initialize shadow effects using SVGRenderer
   * @private
   */
  _initializeShadow() {
    const config = this.config;
    const shadowId = "inner-shadow-" + (config.id || config.classId);
    this.renderer.createShadowFilter(shadowId, {
      verticalOffset: config.shadowVerticalOffset,
      size: config.shadowSize,
      opacity: config.shadowOpacity
    });
    const elementsToShadow = [];
    if (this.canvas.gauge) elementsToShadow.push(this.canvas.gauge);
    if (this.canvas.level) elementsToShadow.push(this.canvas.level);
    this.renderer.applyShadowToElements(shadowId, elementsToShadow);
    this.shadowId = shadowId;
  }
  /**
   * Start initial gauge animation
   * @private
   */
  _startInitialAnimation() {
    const config = this.config;
    if (config.startAnimationTime <= 0) {
      this._drawLevel(config.value);
      if (config.pointer) {
        this._drawPointer();
      }
      if (config.onAnimationEnd) {
        config.onAnimationEnd();
      }
      return;
    }
    let fromValue;
    if (config.differential) {
      fromValue = (config.max + config.min) / 2;
    } else if (config.reverse) {
      fromValue = config.max;
    } else {
      fromValue = config.min;
    }
    this.animator.animate({
      fromValue,
      toValue: config.value,
      duration: config.startAnimationTime,
      easing: config.startAnimationType,
      onUpdate: /* @__PURE__ */ __name((currentValue) => {
        this._drawLevel(currentValue);
        if (config.pointer) {
          this._updatePointer(currentValue);
        }
      }, "onUpdate"),
      onCounterUpdate: config.counter ? (currentValue) => {
        this._updateCounterText(currentValue);
      } : null,
      onComplete: config.onAnimationEnd
    });
  }
  /**
   * Update pointer position during animation
   * @private
   */
  _updatePointer(value) {
    if (this.canvas.pointer) {
      this.canvas.pointer.remove();
    }
    const originalValue = this.config.value;
    this.config.value = value;
    this._drawPointer();
    this.config.value = originalValue;
  }
  /**
   * Update counter text during animation
   * @private
   */
  _updateCounterText(value) {
    const config = this.config;
    let displayValue = value;
    if (config.textRenderer && config.textRenderer(displayValue) !== false) {
      displayValue = config.textRenderer(displayValue);
    } else {
      displayValue = this._formatDisplayText(value, config, "value");
    }
    if (this.canvas.value && !this.config.hideValue) {
      this.canvas.value.attr({ text: displayValue });
    }
  }
  /**
   * Get current display value for animation starting point
   * @private
   */
  _getCurrentDisplayValue() {
    return this.config.value || this.config.min;
  }
};
__name(_JustGage, "JustGage");
var JustGage = _JustGage;

// package-json:../package.json
var package_default = { version: "2.0.1" };

// src/index.js
JustGage.VERSION = package_default.version;
var index_default = JustGage;
export {
  JustGage,
  index_default as default
};
//# sourceMappingURL=justgage.esm.js.map
