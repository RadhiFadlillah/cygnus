var template = `
<div id="page-storage">
    <div class="page-header">
        <a title="Back to list" v-if="selectedFile !== ''" @click="selectedFile = ''">
            <i class="fas fa-fw fa-arrow-left"></i>
        </a>
        <p>{{headerTitle}}</p>
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
                <a v-for="time in files" 
                    @click="selectFile(date, time)" 
                    :class="{active: date+'-'+time === selectedFile}">{{time}}</a>
            </div>
        </div>
    </div>
    <video id="video-viewer" ref="videoViewer" muted controls v-show="selectedFile !== ''"></video>
    <p class="empty-message" v-if="!loading && listIsEmpty">No saved videos yet :(</p>
    <div class="loading-overlay" v-if="loading"><i class="fas fa-fw fa-spin fa-spinner"></i></div>
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
            player: null,
            fileGroups: {},
            selectedDate: "",
            selectedFile: "",
            loading: false,
        }
    },
    computed: {
        headerTitle() {
            if (this.selectedFile === "") return "Storage";
            return this.selectedFile;
        },
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
            this.selectedFile = `${date}-${time}`;
        },
        loadListFile() {
            this.fileGroups = {};
            this.selectedDate = "";
            this.selectedFile = "";
            this.loading = true;

            fetch("/api/storage")
                .then(response => {
                    if (!response.ok) throw response;
                    return response.json();
                })
                .then(json => {
                    this.fileGroups = json;
                    this.loading = false;
                })
                .catch(err => {
                    this.loading = false;
                    err.text().then(msg => {
                        this.showErrorDialog(`${msg} (${err.status})`);
                    })
                });
        }
    },
    watch: {
        selectedFile(val) {
            this.player.detachMedia();

            if (val !== "") {
                this.player.attachMedia(this.$refs.videoViewer);
                this.player.loadSource(`/video/${val}/playlist`);
                this.$refs.videoViewer.play();
            }
        }
    },
    mounted() {
        this.loadListFile();
        if (Hls.isSupported()) {
            this.player = new Hls();
        }
    }
}