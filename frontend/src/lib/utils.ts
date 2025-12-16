import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import { useEffect, useState } from "react";

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs))
}

export function formatDate(date: string | Date): string {
    return new Intl.DateTimeFormat('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    }).format(new Date(date));
}

export function formatCost(cost: number | string | undefined | null): string {
    const val = Number(cost || 0);
    if (isNaN(val)) return '$0.0000';
    return `$${val.toFixed(4)}`;
}

export function formatLatency(ms: number | string | undefined | null): string {
    const val = Number(ms || 0);
    if (isNaN(val)) return '0ms';

    if (val < 1000) {
        return `${Math.round(val)}ms`;
    }
    return `${(val / 1000).toFixed(2)}s`;
}

export function formatTokens(tokens: number | string | undefined | null): string {
    const val = Number(tokens || 0);
    if (isNaN(val)) return '0';

    if (val >= 1000000) {
        return `${(val / 1000000).toFixed(1)}M`;
    }
    if (val >= 1000) {
        return `${(val / 1000).toFixed(1)}K`;
    }
    return String(Math.round(val));
}

export function useDebounce<T>(value: T, delay: number) {
    const [debouncedValue, setDebouncedValue] = useState(value);

    useEffect(() => {
        const handler = setTimeout(() => {
            setDebouncedValue(value);
        }, delay);
        return () => {
            clearTimeout(handler);
        };
    }, [value, delay]);

    return debouncedValue;
}