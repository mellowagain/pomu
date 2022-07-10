<script lang="ts">
  import "carbon-components-svelte/css/g100.css";
  import VideoInput from "./lib/VideoInput.svelte";
  import Logo from "./lib/Logo.svelte";
  import Heading from "./lib/Heading.svelte";
  import ArchiveButtons from "./lib/ArchiveButtons.svelte";
  import AuthNotification from "./lib/AuthNotification.svelte";
</script>

<main>
    <div class="content" align="center">
        <Logo />
        <Heading />

        <VideoInput />

        <ArchiveButtons />

        <AuthNotification />
    </div>
</main>

<style>
    main {
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .content {
        width: 80%;
    }
</style>
