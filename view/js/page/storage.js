var template = `
<div id="page-storage">
    <div class="page-header">
        <p>Storage {{selectedFile}}</p>
        <a title="Save video" v-if="selectedFile !== ''" :href="downloadURL" target="_blank" download>
            <i class="fas fa-fw fa-save"></i>
        </a>
        <a href="#" title="Refresh storage" @click="loadListFile">
            <i class="fas fa-fw fa-sync-alt"></i>
        </a>
    </div>
    <div class="file-list" v-if="!listIsEmpty">
        <div v-for="(files, date) in fileGroups" class="file-group" :class="{expanded: selectedDate === date}">
            <a class="file-group-parent" @click="toggleFileGroup(date)">{{date}}</a>
            <div class="file-group-children">
                <a v-for="time in files" @click="selectFile(date, time)">{{time}}</a>
            </div>
        </div>
    </div>
    <div class="video-container">
    <video id="video-viewer" class="cygnus-video video-js" controls preload="auto">
            <p class="vjs-no-js">
                To view this video please enable JavaScript, and consider upgrading to a web browser that
                <a href="https://videojs.com/html5-video-support/" target="_blank">supports HTML5 video</a>
            </p>
        </video>
    </div>
    <p class="empty-message" v-if="!loading && listIsEmpty">No saved videos yet :(</p>
    <div class="loading-spinner" v-if="loading"><i class="fas fa-fw fa-spin fa-spinner"></i></div>
    <cygnus-dialog v-bind="dialog"/>
</div>`;

import cygnusDialog from "../component/dialog.js";
import basePage from "./base.js";

export default {
    template: template,
    mixins: [basePage],
    components: {
        cygnusDialog
    },
    data() {
        return {
            player: {},
            fileGroups: {},
            selectedDate: "",
            selectedFile: "",
            loading: false,
        }
    },
    computed: {
        downloadURL() {
            return `/video/${this.selectedFile}`;
        },
        listIsEmpty() {
            return Object.getOwnPropertyNames(this.fileGroups).length <= 1;
        }
    },
    methods: {
        toggleFileGroup(date) {
            if (this.selectedDate === date) {
                this.selectedDate = "";
            } else {
                this.selectedDate = date;
            }
        },
        selectFile(date, time) {
            this.selectedFile = date + "-" + time;
            this.player.src({
                src: `/video/${this.selectedFile}/playlist`,
                type: "application/x-mpegURL"
            });
            this.player.play();
        },
        loadListFile() {
            this.fileGroups = {};
            this.selectedDate = "";
            this.selectedFile = "";
            this.loading = true;

            fetch("/api/storage")
                .then(response => {
                    return response.json();
                })
                .then(json => {
                    this.fileGroups = json;
                    this.loading = false;
                })
                .catch(err => {
                    this.loading = false;
                    this.showErrorDialog(err.message);
                });
        }
    },
    watch: {
        selectedFile(val) {
            if (val === "") this.player.hide();
            else this.player.show();
        }
    },
    mounted() {
        this.player = videojs("video-viewer");
        this.player.hide();
        this.loadListFile();
    }
}