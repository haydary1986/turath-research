"use client";

import { useState, use } from "react";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Sparkles, Users, Target, Loader2 } from "lucide-react";

interface CollabRec {
  researcher_name: string;
  match_score: number;
  reason_ar: string;
  potential_topics: string[];
}

interface Strategy {
  recommended_journals: { name: string; reason: string }[];
  topics_to_focus: string[];
  coauthor_strategy: string;
  action_items: string[];
  summary_ar: string;
}

type Props = { params: Promise<{ id: string }> };

export default function ResearcherInsightsPage({ params }: Props) {
  const { id } = use(params);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">تحليلات الذكاء الاصطناعي</h1>
        <p className="text-sm text-muted-foreground">توصيات ونصائح مخصصة بواسطة DeepSeek</p>
      </div>

      <Tabs defaultValue="collaboration">
        <TabsList>
          <TabsTrigger value="collaboration">
            <Users className="me-2 size-4" />
            توصيات التعاون
          </TabsTrigger>
          <TabsTrigger value="strategy">
            <Target className="me-2 size-4" />
            استراتيجية النشر
          </TabsTrigger>
        </TabsList>

        <TabsContent value="collaboration" className="mt-4">
          <CollaborationPanel researcherID={id} />
        </TabsContent>

        <TabsContent value="strategy" className="mt-4">
          <StrategyPanel researcherID={id} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function CollaborationPanel({ researcherID }: { researcherID: string }) {
  const [recs, setRecs] = useState<CollabRec[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function analyze() {
    setLoading(true);
    setError("");
    try {
      const res = await api.get<{ data: CollabRec[] }>(`/api/v1/ai/collaboration/${researcherID}`);
      setRecs(res.data?.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-4">
      <Button onClick={analyze} disabled={loading}>
        {loading ? <Loader2 className="size-4 animate-spin" /> : <Sparkles className="size-4" />}
        <span className="ms-2">{loading ? "جاري التحليل..." : "تحليل فرص التعاون"}</span>
      </Button>

      {error && <div className="rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>}

      {recs.map((rec, i) => (
        <Card key={i}>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">{rec.researcher_name}</CardTitle>
              <Badge>{(rec.match_score * 100).toFixed(0)}% توافق</Badge>
            </div>
          </CardHeader>
          <CardContent className="space-y-2">
            <p className="text-sm">{rec.reason_ar}</p>
            <div className="flex flex-wrap gap-1">
              {rec.potential_topics?.map((t, j) => (
                <Badge key={j} variant="outline" className="text-xs">{t}</Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      ))}

      {!loading && recs.length === 0 && !error && (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            <Users className="mx-auto size-8" />
            <p className="mt-3">اضغط الزر لتحليل فرص التعاون البحثي</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function StrategyPanel({ researcherID }: { researcherID: string }) {
  const [strategy, setStrategy] = useState<Strategy | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function analyze() {
    setLoading(true);
    setError("");
    try {
      const res = await api.get<{ data: Strategy }>(`/api/v1/ai/strategy/${researcherID}`);
      setStrategy(res.data?.data || null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-4">
      <Button onClick={analyze} disabled={loading}>
        {loading ? <Loader2 className="size-4 animate-spin" /> : <Sparkles className="size-4" />}
        <span className="ms-2">{loading ? "جاري التحليل..." : "تحليل استراتيجية النشر"}</span>
      </Button>

      {error && <div className="rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>}

      {strategy && (
        <div className="space-y-4">
          <Card>
            <CardHeader><CardTitle className="text-sm">الملخص</CardTitle></CardHeader>
            <CardContent><p className="text-sm">{strategy.summary_ar}</p></CardContent>
          </Card>

          {strategy.recommended_journals?.length > 0 && (
            <Card>
              <CardHeader><CardTitle className="text-sm">المجلات المقترحة</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                {strategy.recommended_journals.map((j, i) => (
                  <div key={i} className="flex items-start gap-2">
                    <Badge variant="outline" className="shrink-0">{i + 1}</Badge>
                    <div>
                      <p className="text-sm font-medium">{j.name}</p>
                      <p className="text-xs text-muted-foreground">{j.reason}</p>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          )}

          {strategy.action_items?.length > 0 && (
            <Card>
              <CardHeader><CardTitle className="text-sm">خطوات عملية</CardTitle></CardHeader>
              <CardContent>
                <ul className="space-y-1">
                  {strategy.action_items.map((item, i) => (
                    <li key={i} className="flex items-start gap-2 text-sm">
                      <span className="mt-1 size-1.5 shrink-0 rounded-full bg-primary" />
                      {item}
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          )}

          {strategy.topics_to_focus?.length > 0 && (
            <Card>
              <CardHeader><CardTitle className="text-sm">مواضيع مقترحة</CardTitle></CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  {strategy.topics_to_focus.map((t, i) => (
                    <Badge key={i} variant="secondary">{t}</Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {!loading && !strategy && !error && (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            <Target className="mx-auto size-8" />
            <p className="mt-3">اضغط الزر للحصول على نصائح نشر مخصصة</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
