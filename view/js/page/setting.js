var template = `
<div id="page-setting">
    <div class="page-header">
        <p>Settings</p>
        <a href="#" title="Refresh setting" @click="loadSetting">
            <i class="fas fa-fw fa-sync-alt"></i>
        </a>
    </div>
    <div class="setting-container">
        <details open class="setting-group" id="setting-users">
            <summary>Users</summary>
            <ul>
                <li v-if="users.length === 0">No user registered</li>
                <li v-for="(user, idx) in users">{{user}} <a title="Delete user" @click="showDialogDeleteUser(user, idx)">
                    <i class="fa fas fa-fw fa-trash-alt"></i>
                </a></li>
            </ul>
            <div class="setting-group-footer">
                <a @click="showDialogNewUser">Add new user</a>
            </div>
        </details>
        <details open class="setting-group" id="setting-camera">
            <summary>Camera</summary>
            <div class="setting-group-form">
                <label for="select-resolution">Resolution</label>
                <div class="setting-group-select">
                    <select id="select-resolution" v-model="camera.resolution">
                        <option>640x480</option>
                        <option>800x600</option>
                        <option>960x720</option>
                        <option>1024x768</option>
                        <option>1280x960</option>
                        <option>1296x972</option>
                        <option>1440x1080</option>
                    </select>
                </div>
                <label for="input-fps">Framerate</label>
                <input type="number" id="input-fps" v-model="camera.fps"/>
                <label for="select-rotation">Rotation</label>
                <div class="setting-group-select">
                    <select id="select-rotation" v-model="camera.rotation">
                        <option>0</option>
                        <option>90</option>
                        <option>180</option>
                        <option>270</option>
                    </select>
                </div>
            </div>
            <div class="setting-group-footer">
                <a @click="showDialogSaveCamera">Save Setting</a>
                <a @click="showDialogReboot">Reboot Camera</a>
            </div>
        </details>
    </div>
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
            users: [],
            camera: {},
            loading: false,
        }
    },
    methods: {
        loadSetting() {
            this.loading = true;

            fetch("/api/setting")
                .then(response => {
                    if (!response.ok) throw response;
                    return response.json();
                })
                .then(json => {
                    this.users = json.users;
                    this.camera = json.camera;
                    this.loading = false;
                })
                .catch(err => {
                    this.loading = false;
                    err.text().then(msg => {
                        this.showErrorDialog(`${msg} (${err.status})`);
                    })
                });
        },
        showDialogNewUser() {
            this.showDialog({
                title: "New User",
                content: "Input new user's data :",
                fields: [{
                    name: "username",
                    label: "Username",
                    value: "",
                }, {
                    name: "password",
                    label: "Password",
                    type: "password",
                    value: "",
                }, {
                    name: "repeat",
                    label: "Repeat password",
                    type: "password",
                    value: "",
                }],
                mainText: "OK",
                secondText: "Cancel",
                mainClick: (data) => {
                    if (data.username === "") {
                        this.showErrorDialog("Username must not empty");
                        return;
                    }

                    if (data.password === "") {
                        this.showErrorDialog("Password must not empty");
                        return;
                    }

                    if (data.password !== data.repeat) {
                        this.showErrorDialog("Password does not match");
                        return;
                    }

                    this.dialog.loading = true;
                    fetch("/api/user", {
                            method: "post",
                            body: JSON.stringify(data),
                            headers: {
                                "Content-Type": "application/json",
                            },
                        })
                        .then(response => {
                            if (!response.ok) throw response;
                            return response;
                        })
                        .then(() => {
                            this.dialog.loading = false;
                            this.dialog.visible = false;
                            this.users.push(data.username);
                            this.users.sort();
                        })
                        .catch(err => {
                            this.dialog.loading = false;
                            err.text().then(msg => {
                                this.showErrorDialog(`${msg} (${err.status})`);
                            })
                        });
                }
            });
        },
        showDialogDeleteUser(username, idx) {
            this.showDialog({
                title: "Delete User",
                content: `Delete user "${username} ?`,
                mainText: "Yes",
                secondText: "No",
                mainClick: () => {
                    this.dialog.loading = true;
                    fetch(`/api/user/${username}`, { method: "delete" })
                        .then(response => {
                            if (!response.ok) throw response;
                            return response;
                        })
                        .then(() => {
                            this.dialog.loading = false;
                            this.dialog.visible = false;
                            this.users.splice(idx, 1);
                        })
                        .catch(err => {
                            this.dialog.loading = false;
                            err.text().then(msg => {
                                this.showErrorDialog(`${msg} (${err.status})`);
                            })
                        });
                }
            });
        },
        showDialogSaveCamera() {
            this.showDialog({
                title: "Camera Setting",
                content: "Save camera setting ?",
                mainText: "Yes",
                secondText: "No",
                mainClick: () => {
                    this.dialog.loading = true;
                    fetch("/api/setting/camera", {
                            method: "post",
                            body: JSON.stringify(this.camera),
                            headers: {
                                "Content-Type": "application/json",
                            },
                        })
                        .then(response => {
                            if (!response.ok) throw response;
                            return response;
                        })
                        .then(() => {
                            setTimeout(() => location.href = "/login", 3500);
                        })
                        .catch(err => {
                            this.dialog.loading = false;
                            err.text().then(msg => {
                                this.showErrorDialog(`${msg} (${err.status})`);
                            })
                        });
                }
            });
        },
        showDialogReboot() {
            this.showDialog({
                title: "Reboot Camera",
                content: "Reboot the camera ?",
                mainText: "Yes",
                secondText: "No",
                mainClick: () => {
                    this.dialog.loading = true;
                    fetch("/api/setting/reboot", {
                            method: "post",
                        })
                        .then(response => {
                            if (!response.ok) throw response;
                            return response;
                        })
                        .then(() => {
                            setTimeout(() => location.href = "/login", 10000);
                        })
                        .catch(err => {
                            this.dialog.loading = false;
                            err.text().then(msg => {
                                this.showErrorDialog(`${msg} (${err.status})`);
                            })
                        });
                }
            });
        }
    },
    mounted() {
        this.loadSetting();
    }
}