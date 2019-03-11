var template = `
<div id="live-page">
    <video id="video-viewer" class="cygnus-video video-js" controls preload="auto">
        <source src="/playlist/live" type="application/vnd.apple.mpegurl">
        <p class="vjs-no-js">
            To view this video please enable JavaScript, and consider upgrading to a web browser that
            <a href="https://videojs.com/html5-video-support/" target="_blank">supports HTML5 video</a>
        </p>
    </video>
</div>`;

export default {
    template: template,
    mounted() {
        videojs("video-viewer", { autoplay: true });
    }
}