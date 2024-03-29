<script lang="ts">
    import {
        Button,
        Dropdown,
        Form,
        FormGroup,
        TextInput,
    } from "carbon-components-svelte";
    import { acceptOnlyOkResponse } from "./api";
    import { showNotification } from "./notifications";
    import { videoInputInfoStore } from "./video";
    import { onDestroy } from "svelte";

    let qualities = [];
    let disabled = true;
    let streamUrl = "";

    let selectedId = "0";

    let disableQualitiesDropdown = true;

    function clearVideoDisplay() {
        qualities = [];
        selectedId = "0";
        disableQualitiesDropdown = true;
        videoInputInfoStore.set(null);
    }

    onDestroy(() => {
        clearVideoDisplay();
    });

    async function resolveQualities(_: any) {
        if (streamUrl.trim().length === 0) {
            clearVideoDisplay();
            return;
        }

        qualities = [];

        await fetchVideoInfo(streamUrl);

        let url = new URL(
            `${window.location.protocol}//${window.location.host}/api/qualities`
        );
        url.searchParams.set("url", streamUrl);

        let items = await fetch(url.toString())
            .then((r) => r.json())
            .catch(console.log);

        for (let item of items) {
            qualities.push({
                id: item.code.toString(),
                text: item.resolution,
                best: item.best,
                code: item.code,
            });

            if (item.best) {
                selectedId = item.code.toString();
            }
        }

        // https://stackoverflow.com/a/69250874/11494565
        qualities.sort((a, b) => a.code - b.code);
        disableQualitiesDropdown = false;
    }

    async function submitForm(event: any) {
        event.preventDefault();

        try {
            await fetch(`/api/submit`, {
                method: "POST",
                body: JSON.stringify({
                    videoUrl: streamUrl,
                    quality: +selectedId,
                }),
            })
                .then(acceptOnlyOkResponse)
                .then((r) => r.json());
        } catch (e) {
            if ("status" in e && "why" in e) {
                showNotification({
                    title:
                        "Failed to submit video (" +
                        e.status +
                        " " +
                        e.statusText +
                        ")",
                    description: "Api returned '" + e.why + "'.",
                    kind: "error",
                    timeout: 5000,
                });
            } else {
                showNotification({
                    title: "Failed to submit video (" + e.status + ")",
                    description: "Unknown error.",
                    kind: "error",
                    timeout: 5000,
                });
            }

            return;
        }

        showNotification({
            title: "Successfully submitted video",
            description: "Recording will begin when the stream starts.",
            kind: "success",
            timeout: 5000,
        });
    }

    async function fetchVideoInfo(url: string) {
        let info = await fetch("https://www.youtube.com/oembed?url=" + url)
            .then((r) => r.json())
            .catch((r) => {
                console.log(r);
                clearVideoDisplay();
            });

        let thumbnailUrl = info.thumbnail_url.replaceAll(
            "hqdefault",
            "maxresdefault"
        );
        videoInputInfoStore.update((_) => ({
            thumbnailUrl,
            title: info.title,
            uploader: info.author_name,
        }));
    }
</script>

<Form on:submit={submitForm}>
    <FormGroup>
        <TextInput
            labelText="Livestream url"
            placeholder="https://youtube.com/watch?v=rnVfwYuK8sw"
            on:change={resolveQualities}
            bind:value={streamUrl}
        />

        <Dropdown
            itemToString={(item) =>
                (item.best ? "[BEST] " : "") +
                item.text +
                " (id " +
                item.id +
                ")"}
            disabled={disableQualitiesDropdown}
            titleText="Quality"
            bind:selectedId
            items={qualities}
        />
    </FormGroup>
    <Button type="submit" disabled={streamUrl.trim().length === 0}
        >Add to archive queue</Button
    >
</Form>
