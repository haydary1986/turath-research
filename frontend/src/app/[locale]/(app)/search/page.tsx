"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Search, Users, FileText, BookOpen, Loader2 } from "lucide-react";

interface SearchResult {
  type: "researcher" | "publication" | "journal";
  id: string;
  title: string;
  sub: string;
  score: number;
}

const typeIcons = {
  researcher: Users,
  publication: FileText,
  journal: BookOpen,
};

const typeLabels: Record<string, string> = {
  researcher: "باحث",
  publication: "بحث",
  journal: "مجلة",
};

const typeColors: Record<string, string> = {
  researcher: "bg-status-info-bg text-status-info-fg",
  publication: "bg-status-success-bg text-status-success-fg",
  journal: "bg-status-progress-bg text-status-progress-fg",
};

const typeLinks: Record<string, (id: string) => string> = {
  researcher: (id) => `/researchers/${id}`,
  publication: (id) => `/publications/${id}`,
  journal: () => "#",
};

export default function SearchPage() {
  const ct = useTranslations("common");
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);

  async function handleSearch(q: string) {
    setQuery(q);
    if (q.length < 2) {
      setResults([]);
      setSearched(false);
      return;
    }

    setLoading(true);
    setSearched(true);
    try {
      const res = await api.get<SearchResult[]>(`/api/v1/search?q=${encodeURIComponent(q)}&limit=30`);
      setResults(res.data || []);
    } catch {
      setResults([]);
    } finally {
      setLoading(false);
    }
  }

  // Debounce
  let timer: ReturnType<typeof setTimeout>;
  function onInput(val: string) {
    setQuery(val);
    clearTimeout(timer);
    timer = setTimeout(() => handleSearch(val), 300);
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{ct("search")}</h1>

      <div className="relative">
        <Search className="absolute start-4 top-1/2 size-5 -translate-y-1/2 text-muted-foreground" />
        {loading && <Loader2 className="absolute end-4 top-1/2 size-5 -translate-y-1/2 animate-spin text-muted-foreground" />}
        <Input
          placeholder="ابحث عن باحث، بحث، أو مجلة..."
          value={query}
          onChange={(e) => onInput(e.target.value)}
          className="h-12 ps-12 text-lg"
          autoFocus
        />
      </div>

      {searched && !loading && results.length === 0 && (
        <p className="text-center text-muted-foreground py-8">{ct("noResults")}</p>
      )}

      {results.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>{results.length} نتيجة</span>
              {["researcher", "publication", "journal"].map((t) => {
                const count = results.filter((r) => r.type === t).length;
                if (count === 0) return null;
                return (
                  <Badge key={t} variant="outline" className="text-xs">
                    {typeLabels[t]}: {count}
                  </Badge>
                );
              })}
            </div>
          </CardHeader>
          <CardContent className="space-y-1">
            {results.map((r) => {
              const Icon = typeIcons[r.type] || FileText;
              return (
                <Link
                  key={`${r.type}-${r.id}`}
                  href={typeLinks[r.type]?.(r.id) || "#"}
                  className="flex items-center gap-3 rounded-lg p-3 transition-colors hover:bg-muted"
                >
                  <div className={`rounded-lg p-2 ${typeColors[r.type]}`}>
                    <Icon className="size-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{r.title}</p>
                    {r.sub && <p className="text-xs text-muted-foreground truncate">{r.sub}</p>}
                  </div>
                  <Badge variant="outline" className="shrink-0 text-xs">
                    {typeLabels[r.type]}
                  </Badge>
                </Link>
              );
            })}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
