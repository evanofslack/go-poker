import { useEffect, useState } from "react";
import { Button, TextInput } from "@mantine/core";
import { AuditFilter } from "../../lib/admin";
import Segmented from "../Segmented";
import AuditList from "./AuditList";

type By = "ip" | "name" | "person";

type lookupProps = {
    filter: AuditFilter | null;
    onSearch: (filter: AuditFilter) => void;
    onTable: (name: string) => void;
};

// Lookup searches the audit across every table, open or closed, by address,
// name or person id: to find one person behind several names, or several
// people behind one.
export default function Lookup({ filter, onSearch, onTable }: lookupProps) {
    const [by, setBy] = useState<By>("ip");
    const [value, setValue] = useState("");

    // a link elsewhere on the page searches too
    useEffect(() => {
        if (!filter) return;
        const b: By = filter.person ? "person" : filter.name ? "name" : "ip";
        setBy(b);
        setValue(filter[b] ?? "");
    }, [filter]);

    return (
        <div className="flex flex-col gap-4">
            <form
                className="flex flex-wrap items-end gap-3"
                onSubmit={(e) => {
                    e.preventDefault();
                    if (value.trim()) onSearch({ [by]: value.trim() });
                }}
            >
                <Segmented
                    legend="by"
                    name="lookup-by"
                    options={[
                        { label: "address", value: "ip" as By },
                        { label: "name", value: "name" as By },
                        { label: "person id", value: "person" as By },
                    ]}
                    value={by}
                    onChange={setBy}
                />
                <TextInput
                    aria-label="address, name or person id"
                    className="min-w-48 flex-1"
                    value={value}
                    onChange={(e) => setValue(e.currentTarget.value)}
                />
                <Button type="submit" variant="filled">
                    search
                </Button>
            </form>
            {filter && <AuditList filter={filter} onTable={onTable} onLookup={onSearch} />}
        </div>
    );
}
