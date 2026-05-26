export function toTimeElapsed(secondsIn) {
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
