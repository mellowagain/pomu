<script lang="ts">
    import { DataTable, DataTableSkeleton, InlineNotification, TooltipIcon } from "carbon-components-svelte";
    import { showNotification } from "./notifications";
    import type { Statistic } from "./stats";
    import dayjs from "dayjs";
    import duration from "dayjs/plugin/duration";
    import relativeTime from "dayjs/plugin/relativeTime";
    import { humanizeFileSize } from "./video";
    import { Information } from "carbon-icons-svelte";

    dayjs.extend(duration);
    dayjs.extend(relativeTime);

    async function requestStats(): Promise<Array<Statistic>> {
        try {
            let result = await fetch("/api/stats").then((r) => r.json());

            return [
                {
                    id: 0,
                    key: "Videos",
                    value: result.videoAmount,
                    description: "Amount of livestreams stored"
                } as Statistic,
                {
                    id: 1,
                    key: "File Size",
                    value: humanizeFileSize(result.totalFileSize),
                    description: "Total file size of all stored livestreams"
                } as Statistic,
                {
                    id: 2,
                    key: "Length",
                    value: result.totalLength,
                    description: "Total length of all stored livestreams"
                } as Statistic,
                {
                    id: 3,
                    key: "Channels",
                    value: result.uniqueChannels,
                    description: "Unique channels for which livestreams were stored"
                } as Statistic,
                {
                    id: 4,
                    key: "S3 Bill",
                    value: "$" + Number((result.s3BillPerMonth).toFixed(3)),
                    description: "Storage costs per month"
                } as Statistic,
            ];
        } catch (e) {
            console.log(e);
            showNotification({
                title: "Failed to get stats",
                description: e.text,
                kind: "error",
                timeout: 5000,
            });
        }
    }
</script>

<div>
    {#await requestStats()}
        <DataTableSkeleton
            headers={[
                { value: "Statistic" },
                { value: "Value" },
                { value: "Description" }
            ]}
            rows={4}
        />
    {:then rows}
        <DataTable
            title="Statistics"
            description="Instance: {window.location === "pomu.app" ? "Production" : window.location.hostname.split(".")[0]}"
            headers={[
                { key: "key", value: "Statistic" },
                { key: "value", value: "Value" },
                { key: "description", value: "Description" }
            ]}
            rows={rows}
        >
            <svelte:fragment slot="cell" let:row let:cell>
                {#if cell.key === "value" && row.id === 2}
                    {dayjs.duration(cell.value, "seconds").humanize()}

                    <TooltipIcon
                            icon={Information}
                            tooltipText={dayjs.duration(cell.value, "seconds").format("Y [years] M [months] D [days] H [hours] m [minutes] s [seconds]")}
                    />
                {:else}
                    {cell.value}
                {/if}
            </svelte:fragment>
        </DataTable>
    {/await}
</div>
