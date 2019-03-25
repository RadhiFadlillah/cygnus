var template = `
<div id="page-live">
    <h1 class="page-header">
        Live Stream
    </h1>
    <div class="video-container">
        <video id="live-viewer" class="cygnus-video video-js" controls preload="auto">
            <source src="/live/playlist" type="application/x-mpegURL">
            <p class="vjs-no-js">
                To view this video please enable JavaScript, and consider upgrading to a web browser that
                <a href="https://videojs.com/html5-video-support/" target="_blank">supports HTML5 video</a>
            </p>
        </video>
    </div>
</div>`;

export default {
    template: template,
    mounted() {
        videojs("live-viewer", { autoplay: true });
    }
}