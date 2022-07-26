<script lang="ts">
    import {
        Header,
        HeaderAction,
        HeaderNavItem,
        HeaderPanelLink,
        HeaderUtilities,
        ImageLoader,
        SkipToContent,
    } from "carbon-components-svelte";
    import { user } from "./auth";
    import NavAvatar from "./NavAvatar.svelte";

    let isOpen = false;
</script>

<div>
    <Header company="Pomu.app" href="/">
        <svelte:fragment slot="skip-to-content">
            <SkipToContent />
        </svelte:fragment>
        <HeaderNavItem href="/queue" text="Queue" />
        <HeaderNavItem href="/history" text="History" />
        <HeaderUtilities>
            {#await user then user}
                <HeaderAction icon={NavAvatar}>
                    <ImageLoader src={user.avatar} />
                    <HeaderPanelLink>{user.name}</HeaderPanelLink>
                </HeaderAction>
            {:catch e}
                <HeaderNavItem href="/login" text="Login" />
            {/await}
        </HeaderUtilities>
    </Header>
</div>

<style></style>
