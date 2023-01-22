<script lang="ts">
    import {
        Button, ButtonSet,
        Header,
        HeaderAction,
        HeaderGlobalAction,
        HeaderNavItem,
        HeaderPanelLink,
        HeaderUtilities,
        ImageLoader, Modal,
        SkipToContent,
    } from "carbon-components-svelte";
    import { currentPage, Page } from "./app";
    import { user } from "./api";
    import NavAvatar from "./NavAvatar.svelte";
    import {LogoDiscord, LogoGithub, LogoTwitter, LogoYoutube} from "carbon-icons-svelte";

    currentPage.subscribe(value => {
        switch (value) {
            case Page.Video:
                history.pushState({currentPage: value}, "", `${window.location.origin}`);
                break;
            case Page.Queue:
                history.pushState({currentPage: value}, "", `${window.location.origin}/queue`);
                break;
            case Page.History:
                history.pushState({currentPage: value}, "", `${window.location.origin}/history`);
                break;
            default:
                // TODO: Display 404
                break;
        }
    });

    window.onpopstate = function(event) {
        currentPage.set(event.state.currentPage);
        console.debug(`location: ${document.location}, state: ${JSON.stringify(event.state)}`);
    }

    let loginModalOpen = false;
</script>

<div>
    <Header
        company="Pomu.app"
        platformName="Dev"
        on:click={(_) => currentPage.update((_) => Page.Video)}
    >
        <svelte:fragment slot="skip-to-content">
            <SkipToContent />
        </svelte:fragment>
        <HeaderNavItem
            on:click={(_) => currentPage.update((_) => Page.Queue)}
            text="Queue"
        />
        <HeaderNavItem
            on:click={(_) => currentPage.update((_) => Page.History)}
            text="History"
        />
        <HeaderUtilities>
            <HeaderGlobalAction
                aria-label="GitHub"
                icon={LogoGithub}
                on:click={() => window.open("https://github.com/mellowagain/pomu", "_blank")}
            />

            {#await user then user}
                <HeaderAction icon={NavAvatar}>
                    <ImageLoader src={user.avatar} />
                    <HeaderPanelLink>{user.name}</HeaderPanelLink>
                </HeaderAction>
            {:catch e}
                <HeaderNavItem on:click={() => (loginModalOpen = true)} text="Login" />
            {/await}
        </HeaderUtilities>
    </Header>
</div>

<Modal passiveModal bind:open={loginModalOpen} size="xs" modalHeading="Login using" on:open on:close>
    <ButtonSet stacked>
        <Button kind="tertiary" icon={LogoDiscord} href="/oauth/discord">Discord</Button>
        <Button kind="tertiary" icon={LogoTwitter} href="/oauth/twitter">Twitter</Button>
        <Button kind="tertiary" icon={LogoYoutube} disabled>YouTube</Button>
    </ButtonSet>
</Modal>
