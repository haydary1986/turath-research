"use client";

import { useState } from "react";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { TrendingUp, TrendingDown, Minus, Sparkles, Loader2 } from "lucide-react";
import { Alert } from "@/components/ui/alert";

interface TrendingTopic {
  topic_ar: string;
  topic_en: string;
  direction: "rising" | "declining" | "stable";
  score: number;
  researchers: number;
  publications: number;
}

interface AIResponse {
  cached: boolean;
  data: TrendingTopic[];
}

const directionIcons = {
  rising: TrendingUp,
  declining: TrendingDown,
  stable: Minus,
};

const directionColors = {
  rising: "bg-status-success-bg text-status-success-fg",
  declining: "bg-status-danger-bg text-status-danger-fg",
  stable: "bg-status-pending-bg text-status-pending-fg",
};

const directionLabels = {
  rising: "صاعد",
  declining: "متراجع",
  stable: "مستقر",
};

export default function TrendingTopicsPage() {
  const [topics, setTopics] = useState<TrendingTopic[]>([]);
  const [loading, setLoading] = useState(false);
  const [cached, setCached] = useState(false);
  const [error, setError] = useState("");

  async function analyze() {
    setLoading(true);
    setError("");
    try {
      const res = await api.get<AIResponse>("/api/v1/ai/trends");
      if (res.data) {
        setTopics(res.data.data || []);
        setCached(res.data.cached);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Analysis failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">المواضيع الرائجة</h1>
          <p className="text-sm text-muted-foreground">
            تحليل AI لاتجاهات البحث في المؤسسة
          </p>
        </div>
        <Button onClick={analyze} disabled={loading}>
          {loading ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Sparkles className="size-4" />
          )}
          <span className="ms-2">{loading ? "جاري التحليل..." : "تحليل الاتجاهات"}</span>
        </Button>
      </div>

      {error && (
        <Alert variant="danger">{error}</Alert>
      )}

      {cached && (
        <Alert variant="info">نتائج محفوظة مسبقاً (صالحة لمدة 7 أيام)</Alert>
      )}

      {topics.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2">
          {topics.map((topic, i) => {
            const Icon = directionIcons[topic.direction] || Minus;
            const color = directionColors[topic.direction] || directionColors.stable;
            return (
              <Card key={i}>
                <CardHeader className="flex flex-row items-start justify-between pb-3">
                  <div>
                    <CardTitle className="text-base">{topic.topic_ar}</CardTitle>
                    <p className="text-sm text-muted-foreground">{topic.topic_en}</p>
                  </div>
                  <div className={`rounded-lg p-2 ${color}`}>
                    <Icon className="size-4" />
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-3">
                    <Badge variant={topic.direction === "rising" ? "default" : "secondary"}>
                      {directionLabels[topic.direction]}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {topic.researchers} باحث | {topic.publications} بحث
                    </span>
                    <div className="ms-auto text-sm font-bold">
                      {(topic.score * 100).toFixed(0)}%
                    </div>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      {!loading && topics.length === 0 && !error && (
        <Card>
          <CardContent className="py-12 text-center">
            <Sparkles className="mx-auto size-10 text-muted-foreground" />
            <p className="mt-4 text-muted-foreground">
              اضغط «تحليل الاتجاهات» لبدء تحليل AI للمواضيع البحثية الرائجة
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
