var template = `
<div id="page-live">
    <h1 class="page-header">
        Live Stream
    </h1>
    <video id="live-viewer" ref="liveViewer" autoplay muted controls>
    </video>
</div>`;

export default {
    template: template,
    mounted() {
        if (Hls.isSupported()) {
            var hls = new Hls({
                liveSyncDurationCount: 0,
                liveMaxLatencyDurationCount: 10,
                liveDurationInfinity: true
            });

            hls.loadSource("/live/playlist");
            hls.attachMedia(this.$refs.liveViewer);
            hls.on(Hls.Events.MANIFEST_PARSED, () => {
                this.$refs.liveViewer.play();
            });
        }
    }
}