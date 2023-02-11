<script lang="ts">
    import {
        Column,
        Dropdown,
        Grid,
        InlineNotification,
        NotificationActionButton,
        Pagination,
        PaginationSkeleton,
        Row, Search,
    } from "carbon-components-svelte";
    import SkeletonVideoEntry from "./SkeletonVideoEntry.svelte";
    import type { HistoryResponse, VideoInfo } from "./video";
    import VideoEntry from "./VideoEntry.svelte";
    import { onDestroy } from "svelte";
    import type { SearchMetadata } from "./search";
    import { delay } from "./api.js";
    import { MeiliSearch, SearchResponse } from "meilisearch";

    let sorting = "desc";
    let page = 1;
    let limit = 25;

    let searchValue = "";
    let lastSearch: SearchResponse<Partial<VideoInfo>> = null;
    $: offset = (page - 1) * limit;

    let url;
    let apiKey;
    let indexName;

    let abortController = new AbortController();

    async function requestHistory(): Promise<HistoryResponse> {
        let results = await fetch(`/api/history?page=${page - 1}&limit=${limit}&sort=${sorting}`, {
            signal: abortController.signal
        });
        let json: VideoInfo[] = await results.json();

        return {
            totalItems: +results.headers.get("X-Pomu-Pagination-Total"),
            videos: json
        };
    }

    async function requestMeilisearchData(): Promise<SearchMetadata> {
        let results = await fetch("/api/search", {
            signal: abortController.signal
        });
        let json: SearchMetadata = await results.json();

        if (json.enabled) {
            indexName = json.index;
            url = json.url;
            apiKey = json.apiKey;
        }

        return json;
    }

    async function startSearch(): Promise<SearchResponse<Partial<VideoInfo>>> {
        // set the lastSearch to null to allow us to display a skeleton
        lastSearch = null;

        let searchClient = new MeiliSearch({
            host: url,
            apiKey
        });
        let index = searchClient.index(indexName);

        let search: SearchResponse<Partial<VideoInfo>> = await index.search(searchValue, {
            filter: ["finished = true"],
            sort: ["scheduledStart:desc"],
            page: page,
            offset: offset,
            hitsPerPage: limit,
        }, {
            signal: abortController.signal
        });

        // fix up download urls (as they are not populated, hence the `Partial` in `Partial<VideoInfo>`
        search.hits.forEach((part, index, array) => {
            array[index].downloadUrl = `/api/download/${part.id}/video`;
        });

        lastSearch = search;
        return search;
    }

    // small wrapper function to re-assign `history` and force svelte to re-fetch the data.
    // this is used by basically every button below
    function refreshData() {
        history = requestHistory();
    }

    onDestroy(() => abortController.abort());

    let history = requestHistory();
</script>

<Grid>
    <Row>
        <Column>
            <h1>History</h1>
        </Column>
        <Column></Column>
        <Column style="display: flex; align-items: flex-end;">
            {#await requestMeilisearchData()}
                <Search skeleton />
            {:then params}
                {#if params.enabled}
                    <Search
                        size="lg"
                        bind:value={searchValue}
                        on:keyup={delay(startSearch, 500)}
                    />
                {/if}
            {:catch error}
                <InlineNotification
                    lowContrast
                    hideCloseButton
                    kind="error"
                    title="Failed to load search bar:"
                    subtitle={error}
                />
            {/await}
        </Column>
        <Column>
            {#if searchValue.length === 0}
                <Dropdown
                    titleText="Sorting"
                    bind:selectedId={sorting}
                    items={[
                        { id: "asc", text: "Oldest" },
                        { id: "desc", text: "Newest" }
                    ]}
                    on:select={refreshData}
                />
            {:else}
                <Dropdown
                    titleText="Sorting"
                    selectedId="0"
                    items={[
                        { id: "0", text: "Best Match" },
                    ]}
                    disabled
                />
            {/if}
        </Column>
    </Row>
</Grid>

<div class="divider"></div>

{#if searchValue.length === 0}
    {#await history}
        <PaginationSkeleton/>

        {#each Array(limit) as _, i}
            <SkeletonVideoEntry/>
        {/each}

        <PaginationSkeleton/>
    {:then result}
        {#if result.videos.size === 0}
            <InlineNotification
                lowContrast
                hideCloseButton
                kind="info"
                subtitle="No streams have finished recording"
            />
        {:else}
            <Pagination
                totalItems={result.totalItems}
                pageSizes={[25, 50, 75, 100]}
                bind:pageSize={limit}
                bind:page
                on:click:button--previous={refreshData}
                on:click:button--next={refreshData}
            />

            {#each result.videos as info (info.id)}
                <VideoEntry {info}/>
            {/each}

            <Pagination
                totalItems={result.totalItems}
                pageSizes={[25, 50, 75, 100]}
                bind:pageSize={limit}
                bind:page
                on:update={refreshData}
                on:click:button--previous={refreshData}
                on:click:button--next={refreshData}
            />
        {/if}
    {:catch error}
        <InlineNotification
            lowContrast
            hideCloseButton
            kind="error"
            title="Failed to load history:"
            subtitle={error}
        >
            <svelte:fragment slot="actions">
                <NotificationActionButton on:click={refreshData}>Retry</NotificationActionButton>
            </svelte:fragment>
        </InlineNotification>
    {/await}
{:else}
    {#if lastSearch == null}
        <PaginationSkeleton />

        {#each Array(limit) as _, i}
            <SkeletonVideoEntry />
        {/each}

        <PaginationSkeleton />
    {:else}
        {#if lastSearch.hits.length === 0}
            <InlineNotification
                lowContrast
                hideCloseButton
                kind="info"
                subtitle="No results found matching your query"
            />
        {:else}
            <Pagination
                totalItems={lastSearch.estimatedTotalHits ?? lastSearch.totalHits}
                pageSizes={[25, 50, 75, 100]}
                bind:pageSize={limit}
                bind:page
                on:click:button--previous={startSearch}
                on:click:button--next={startSearch}
            />

            {#each lastSearch.hits as info (info.id)}
                <VideoEntry {info}/>
            {/each}

            <Pagination
                totalItems={lastSearch.estimatedTotalHits ?? lastSearch.totalHits}
                pageSizes={[25, 50, 75, 100]}
                bind:pageSize={limit}
                bind:page
                on:update={startSearch}
                on:click:button--previous={startSearch}
                on:click:button--next={startSearch}
            />
        {/if}
    {/if}
{/if}

<style>
    .divider {
        margin-bottom: 1em;
    }
</style>
