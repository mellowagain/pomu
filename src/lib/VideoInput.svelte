<script lang="ts">
    import {
        Button,
        Dropdown,
        Form,
        FormGroup,
        ImageLoader,
        TextInput,
    } from "carbon-components-svelte";

    let qualities = [];
    let disabled = true;
    let streamUrl = "";

    let thumbnailUrl = "";
    let videoTitle = "";
    let videoUploader = "";
    let selectedId = "0";

    let disableQualitiesDropdown = true;

    function clearVideoDisplay() {
        qualities = [];
        thumbnailUrl = "";
        videoTitle = "";
        videoUploader = "";
        selectedId = "0";
        disableQualitiesDropdown = true;
    }

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

        let response = await fetch(`/api/submit`, {
            method: "POST",
            body: JSON.stringify({
                videoUrl: streamUrl,
                quality: +selectedId
            })
        }).then(r => r.json()).catch(r => console.log(r));

        console.log(response);
    }

    async function fetchVideoInfo(url: string) {
        let info = await fetch("https://www.youtube.com/oembed?url=" + url)
            .then((r) => r.json())
            .catch((r) => {
                console.log(r);
                clearVideoDisplay();
            });

        thumbnailUrl = info.thumbnail_url.replaceAll(
            "hqdefault",
            "maxresdefault"
        );
        videoTitle = info.title;
        videoUploader = info.author_name;
    }
</script>

<div>
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
                bind:selectedId={selectedId}
                items={qualities}
            />
        </FormGroup>

        {#if thumbnailUrl.trim().length !== 0 && videoTitle.trim().length !== 0 && videoUploader.trim().length !== 0}
            <ImageLoader src={thumbnailUrl} />
            <p><b>{videoTitle}</b> by <b>{videoUploader}</b></p>
        {/if}

        <Button type="submit" disabled={streamUrl.trim().length === 0}
            >Add to archive queue</Button
        >
    </Form>
</div>

<style>
    div {
        max-width: 50%;
        text-align: center;
    }
</style>
