"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/routing";
import { useAuth } from "@/lib/auth";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { GoogleSignIn } from "@/components/auth/google-sign-in";
import { GraduationCap } from "lucide-react";

export default function LoginPage() {
  const t = useTranslations("auth");
  const router = useRouter();
  const { setAuth } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await api.post<{
        token: string;
        user: {
          id: string;
          email: string;
          name_ar?: string;
          name_en?: string;
          role: string;
          institution_id?: string;
          college_id?: string;
          department_id?: string;
        };
      }>("/api/v1/auth/login", { email, password });
      if (res.data) {
        setAuth(res.data.token, res.data.user);
        router.push("/portal");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-[80vh] items-center justify-center">
      <Card className="w-full max-w-md border-t-4 border-t-primary shadow-lg">
        <CardHeader className="items-center text-center">
          <div className="mx-auto flex size-14 items-center justify-center rounded-2xl bg-primary text-primary-foreground">
            <GraduationCap className="size-7" aria-hidden />
          </div>
          <CardTitle className="mt-3 text-xl">{t("portalName")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("universityName")}</p>
        </CardHeader>
        <CardContent className="space-y-6">
          {error && (
            <div className="rounded bg-red-50 p-3 text-sm text-red-600">
              {error}
            </div>
          )}
          <GoogleSignIn onError={setError} />
          <button
            type="button"
            onClick={() => setShowPassword((v) => !v)}
            className="block w-full text-center text-xs text-muted-foreground hover:underline"
          >
            {showPassword ? t("hidePasswordLogin") : t("passwordLogin")}
          </button>
          {showPassword && (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">{t("email")}</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  dir="ltr"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">{t("password")}</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  dir="ltr"
                />
              </div>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "..." : t("login")}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
