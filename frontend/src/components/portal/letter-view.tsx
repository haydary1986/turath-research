"use client";

import { useEffect, useState } from "react";
import { LETTER_LABELS_AR } from "@/lib/portal/letter";
import QRCode from "qrcode";
import { fetchObjectUrl } from "@/lib/api";

export interface LetterData {
  letter_no: string;
  diwan_no?: string | null;
  issued_date: string;
  recipient_name: string;
  college_name: string;
  department_name: string;
  header_line_1: string;
  header_line_2: string;
  title_text: string;
  body_text: string;
  closing_text: string;
  signer_name: string;
  signer_title: string;
  background_file_id: string | null;
  signature_file_id: string | null;
}

/** Loads a protected image once and releases the temporary URL on unmount. */
function useProtectedImage(fileId: string | null): string | null {
  const [url, setUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!fileId) return;
    let active = true;
    let created: string | null = null;
    fetchObjectUrl(`/api/v1/portal/files/${fileId}`)
      .then((objectUrl) => {
        if (!active) {
          URL.revokeObjectURL(objectUrl);
          return;
        }
        created = objectUrl;
        setUrl(objectUrl);
      })
      .catch(() => undefined);
    return () => {
      active = false;
      if (created) URL.revokeObjectURL(created);
    };
  }, [fileId]);

  return fileId ? url : null;
}

/** Renders the verification URL as a QR data-URL (empty while generating). */
function useQrCode(text: string | undefined): string | null {
  const [url, setUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!text) return;
    let active = true;
    QRCode.toDataURL(text, { margin: 1, width: 160 })
      .then((dataUrl) => active && setUrl(dataUrl))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [text]);

  return text ? url : null;
}

function formatLetterDate(iso: string): string {
  const [y, m, d] = iso.split("-");
  return y && m && d ? `${d}/${m}/${y}` : iso;
}

interface LetterViewProps {
  letter: LetterData;
  /** Public verification link printed at the bottom of the letter. */
  verifyUrl?: string;
}

/** A4 appreciation letter laid over the uploaded letterhead; identical on screen and in print. */
export function LetterView({ letter, verifyUrl }: LetterViewProps) {
  const t = (key: keyof typeof LETTER_LABELS_AR): string => LETTER_LABELS_AR[key];
  const background = useProtectedImage(letter.background_file_id);
  const signature = useProtectedImage(letter.signature_file_id);
  const qr = useQrCode(verifyUrl);

  return (
    <div
      lang="ar"
      dir="rtl"
      className="relative mx-auto overflow-hidden bg-white text-black shadow-lg print:shadow-none"
      style={{ width: "210mm", minHeight: "297mm", fontFamily: "var(--font-cairo), sans-serif" }}
    >
      {background && (
        // eslint-disable-next-line @next/next/no-img-element -- blob URL of an authenticated file
        <img src={background} alt="" className="absolute inset-0 size-full object-fill" />
      )}
      <div className="relative flex min-h-[297mm] flex-col px-[22mm] pb-[28mm] pt-[40mm]">
        <div className="text-center">
          {letter.header_line_1 && <p className="text-2xl font-bold">{letter.header_line_1}</p>}
          {letter.header_line_2 && <p className="text-lg">{letter.header_line_2}</p>}
        </div>

        <div className="mt-8 flex justify-between text-sm">
          <div>
            <p>
              {t("number")}: <span dir="ltr">{letter.diwan_no || letter.letter_no}</span>
            </p>
            {letter.diwan_no && (
              <p className="text-xs text-gray-600">
                {t("internalNo")}: <span dir="ltr">{letter.letter_no}</span>
              </p>
            )}
          </div>
          <p>
            {t("date")}: <span dir="ltr">{formatLetterDate(letter.issued_date)}</span>
          </p>
        </div>

        {letter.title_text && (
          <h2 className="my-8 text-center text-4xl font-bold text-amber-700">{letter.title_text}</h2>
        )}

        <div className="space-y-1 text-lg font-semibold">
          <p>
            {t("to")} / {letter.recipient_name}
          </p>
          {letter.college_name && (
            <p className="text-base font-normal">
              {t("college")}: {letter.college_name}
            </p>
          )}
          {letter.department_name && (
            <p className="text-base font-normal">
              {t("department")}: {letter.department_name}
            </p>
          )}
        </div>

        <p className="mt-6 whitespace-pre-line text-lg leading-loose">{letter.body_text}</p>

        <div className="mt-10 text-center">
          {letter.closing_text && <p className="text-lg font-semibold">{letter.closing_text}</p>}
          {signature && (
            // eslint-disable-next-line @next/next/no-img-element -- blob URL of an authenticated file
            <img src={signature} alt="" className="mx-auto my-2 h-24 object-contain" />
          )}
          {letter.signer_name && <p className="text-lg font-bold">{letter.signer_name}</p>}
          {letter.signer_title && <p>{letter.signer_title}</p>}
        </div>

        {verifyUrl && (
          <div className="mt-auto flex items-end justify-between gap-4 pt-8">
            <p className="text-[10px] text-gray-600">
              {t("verifyAt")}: <span dir="ltr">{verifyUrl}</span>
            </p>
            {qr && (
              // eslint-disable-next-line @next/next/no-img-element -- locally generated data URL
              <img src={qr} alt={t("verifyAt")} className="size-[20mm] shrink-0" />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
