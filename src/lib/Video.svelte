<script lang="ts">
    import VideoEmbed from "./VideoEmbed.svelte";
    import Logo from "./Logo.svelte";
    import VideoInput from "./VideoInput.svelte";
    import Stats from "./Stats.svelte";
    import Heading from "./Heading.svelte";
    import { Column, Row } from "carbon-components-svelte";
</script>

<svelte:head>
    <title>pomu.app</title>
</svelte:head>

<Row>
    <Column>
        <Heading />
        <Logo />
        <Stats />
    </Column>
    <Column>
        <VideoEmbed />
        <VideoInput />
    </Column>
</Row>
