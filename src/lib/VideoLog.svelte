<script lang="ts">
    import {
        Button,
        CodeSnippet,
        InlineNotification,
        Modal,
        NotificationActionButton,
    } from "carbon-components-svelte";
    import { Report } from "carbon-icons-svelte";

    import { onDestroy } from "svelte";

    export let id: string;
    export let downloadUrl: string | undefined;

    onDestroy(() => {
        logCancelled();
    });

    let logOpened: () => void;
    let logCancelled: () => void;
    let waitForLogOpened = new Promise<void>((resolve, reject) => {
        logOpened = resolve;
        logCancelled = reject;
    });

    let getLog = async (method: string) => {
        let result;

        if (downloadUrl != null) {
            result = await fetch(downloadUrl.replace("/video", "/ffmpeg"), {
                method,
            });
        } else {
            // in-progress log
            result = await fetch("/api/logz?id=" + id, { method });
        }

        if (result.status != 200) {
            throw "no log available for " + id;
        }
        return result;
    };

    let hasLog = (async () => {
        try {
            await getLog("HEAD");
            return true;
        } catch (_) {
            return false;
        }
    })();

    let log = async () => {
        await waitForLogOpened;

        let result = await getLog("GET");

        return result
            .text()
            .then((text) =>
                text
                    .replaceAll("frame=", "\nframe=")
                    .replaceAll("[mpegts", "\n[mpegts")
            );
    };

    let logModal = false;
</script>

{#await hasLog}
    <Button
        icon={Report}
        iconDescription="FFMpeg Log"
        kind="tertiary"
        disabled
    />
{:then hasLog}
    <Button
        icon={Report}
        iconDescription="FFMpeg Log"
        kind="tertiary"
        disabled={!hasLog}
        on:click={(_) => (logModal = true)}
    />
{/await}

<Modal
    bind:open={logModal}
    on:open={(_) => logOpened()}
    size="lg"
    passiveModal
    modalHeading={"FFMpeg Log"}
>
    {#await log()}
        <CodeSnippet type="multi" expanded skeleton />
    {:then log}
        <CodeSnippet type="multi" expanded hideCopyButton>
            {log}
        </CodeSnippet>
    {:catch error}
        <InlineNotification
            lowContrast
            hideCloseButton
            kind="error"
            title="Failed to load log:"
            subtitle={error}
        />
    {/await}
</Modal>
