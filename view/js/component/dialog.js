var template = `
<div class="cygnus-dialog-overlay">
    <div class="cygnus-dialog">
        <p class="cygnus-dialog-header">{{title}}</p>
        <p class="cygnus-dialog-content">{{content}}</p>
        <div class="cygnus-dialog-footer">
            <a ref="mainButton" @click="handleAccepted" @keyup.enter="handleAccepted">OK</a>
        </div>
    </div>
</div>`;

export default {
    template: template,
    props: {
        visible: Boolean,
        title: String,
        content: String,
        btnCaption: {
            type: String,
            default: 'OK'
        },
    },
    watch: {
        visible() {
            this.focus();
        }
    },
    methods: {
        handleAccepted() {
            this.$emit("accepted");
        },
        focus() {
            this.$nextTick(() => {
                if (!this.visible) return;
                this.$refs.mainButton.focus();
            })
        }
    }
}