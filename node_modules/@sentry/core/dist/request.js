Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
/** Creates a SentryRequest from an event. */
function sessionToSentryRequest(session, api) {
    var envelopeHeaders = JSON.stringify({
        sent_at: new Date().toISOString(),
    });
    var itemHeaders = JSON.stringify({
        type: 'session',
    });
    return {
        body: envelopeHeaders + "\n" + itemHeaders + "\n" + JSON.stringify(session),
        type: 'session',
        url: api.getEnvelopeEndpointWithUrlEncodedAuth(),
    };
}
exports.sessionToSentryRequest = sessionToSentryRequest;
/** Creates a SentryRequest from an event. */
function eventToSentryRequest(event, api) {
    // since JS has no Object.prototype.pop()
    var _a = event.tags || {}, samplingMethod = _a.__sentry_samplingMethod, sampleRate = _a.__sentry_sampleRate, otherTags = tslib_1.__rest(_a, ["__sentry_samplingMethod", "__sentry_sampleRate"]);
    event.tags = otherTags;
    var useEnvelope = event.type === 'transaction';
    var req = {
        body: JSON.stringify(event),
        type: event.type || 'event',
        url: useEnvelope ? api.getEnvelopeEndpointWithUrlEncodedAuth() : api.getStoreEndpointWithUrlEncodedAuth(),
    };
    // https://develop.sentry.dev/sdk/envelopes/
    // Since we don't need to manipulate envelopes nor store them, there is no
    // exported concept of an Envelope with operations including serialization and
    // deserialization. Instead, we only implement a minimal subset of the spec to
    // serialize events inline here.
    if (useEnvelope) {
        var envelopeHeaders = JSON.stringify({
            event_id: event.event_id,
            sent_at: new Date().toISOString(),
        });
        var itemHeaders = JSON.stringify({
            type: event.type,
            // TODO: Right now, sampleRate may or may not be defined (it won't be in the cases of inheritance and
            // explicitly-set sampling decisions). Are we good with that?
            sample_rates: [{ id: samplingMethod, rate: sampleRate }],
        });
        // The trailing newline is optional. We intentionally don't send it to avoid
        // sending unnecessary bytes.
        //
        // const envelope = `${envelopeHeaders}\n${itemHeaders}\n${req.body}\n`;
        var envelope = envelopeHeaders + "\n" + itemHeaders + "\n" + req.body;
        req.body = envelope;
    }
    return req;
}
exports.eventToSentryRequest = eventToSentryRequest;
//# sourceMappingURL=request.js.map