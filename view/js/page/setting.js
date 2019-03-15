var template = `
<div id="page-setting">
    <h1 class="page-header">
        Settings
    </h1>
    <div v-if="!loading" class="setting-container">
        <details open class="setting-group" id="setting-users">
            <summary>Users</summary>
            <ul>
                <li v-if="users.length === 0">No user registered</li>
                <li v-for="(user, idx) in users">{{user}} <a title="Delete user" @click="showDialogDeleteUser(user, idx)">
                    <i class="fa fas fa-fw fa-trash-alt"></i>
                </a></li>
            </ul>
            <div class="setting-group-footer">
                <a @click="loadData">Refresh</a>
                <a @click="showDialogNewUser">Add new user</a>
            </div>
        </details>
    </div>
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
            users: [],
            loading: false,
        }
    },
    methods: {
        loadData() {
            this.loading = true;

            fetch("/api/user")
                .then(response => {
                    return response.json();
                })
                .then(json => {
                    this.users = json;
                    this.loading = false;
                })
                .catch(err => {
                    this.loading = false;
                    this.showErrorDialog(err.message);
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
                        .then(() => {
                            this.dialog.loading = false;
                            this.dialog.visible = false;
                            this.users.push(data.username);
                            this.users.sort();
                        })
                        .catch(err => {
                            this.dialog.loading = false;
                            this.showErrorDialog(err.message);
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
                        .then(() => {
                            this.dialog.loading = false;
                            this.dialog.visible = false;
                            this.users.splice(idx, 1);
                        })
                        .catch(err => {
                            this.dialog.loading = false;
                            this.showErrorDialog(err.message);
                        });
                }
            });
        }
    },
    mounted() {
        this.loadData();
    }
}