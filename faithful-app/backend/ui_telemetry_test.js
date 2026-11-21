// Static UI contract: any failed telemetry request resets both the UI state and
// remembered trailer, so a newly-attached identical model is detected again.
const fs = require('node:fs');
const assert = require('node:assert/strict');
const source = fs.readFileSync(__dirname + '/web/app.js','utf8');
const marker = 'async function pollTelemetry()';
const poll = source.slice(source.indexOf(marker));
assert(poll.includes("telemetryTrailerKey='';"), 'stale trailer key must clear');
assert(poll.includes("window.refreshEssieuTelemetry({connected:false"), 'connection loss must refresh the status');
assert(poll.includes("telemetryMatchText='En attente de télémétrie';"), 'connection loss must clear stale match label');
console.log('Telemetry UI disconnection contract: OK');
