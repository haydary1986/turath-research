import type { ArticleSet } from "./article-types";

/** English guide articles (mirror of the Arabic set, written for international readers). */
export const ARTICLES_EN: ArticleSet = {
  guide: {
    title: "Predatory Journals: A Practical Guide for Researchers",
    description:
      "What predatory journals are, why they matter for academic promotion, how to tell if a journal is predatory or hijacked, plus a predatory journals list and a free journal check.",
    navLabel: "Predatory journals guide",
    intro:
      "Predatory journals charge authors publication fees and promise fast peer review and indexing, but provide neither genuine review nor trustworthy indexing. This guide gathers what Al-Turath University academics and researchers need before choosing where to publish.",
    sections: [
      {
        heading: "What are predatory journals?",
        paragraphs: [
          "A predatory journal puts revenue ahead of scholarly integrity: it accepts papers within days, asks for fees right after acceptance and advertises impact factors or indexing it does not have. Many use names that differ from a well-known journal by a single word, or claim ties to a country or society they have nothing to do with.",
          "Charging a fee does not make a journal predatory; reputable open-access journals charge fees too. The difference is real peer review, a known editorial board, published policies and indexing you can confirm in the databases themselves.",
        ],
      },
      {
        heading: "Why it matters for promotion",
        paragraphs: [
          "Academic promotion and appreciation letters depend on papers published in reputable journals. A paper in a predatory journal may not count, and it can be hard to publish the same work elsewhere afterwards, besides the damage to the author's and the university's reputation.",
          "That is why Scientific Affairs checks the journal before a paper is accepted for any procedure. The reference is always the lists and instructions of the Ministry of Higher Education and Scientific Research and the decision of Scientific Affairs.",
        ],
      },
      {
        heading: "Quick steps before you submit",
        paragraphs: ["Before sending a paper to a journal you do not know well:"],
        bullets: [
          "Search the predatory journals, predatory publishers and hijacked journals lists in this guide.",
          "Use the journal check to match the name, ISSN or website against the lists and open sources.",
          "Verify Scopus and Web of Science indexing yourself on their official sites, not on the journal's site.",
          "Make sure the website is the journal's real site and not a hijacked copy carrying its name.",
          "If you still have doubts, ask Scientific Affairs before you submit, not after.",
        ],
      },
    ],
    faq: [
      {
        question: "How can I tell if a journal is predatory?",
        answer:
          "Look it up in the predatory journal and publisher lists, watch for warning signs such as very fast acceptance, fees before review and unknown impact metrics, then verify its indexing on the official Scopus and Web of Science sites.",
      },
      {
        question: "Is a listing on a predatory journals list a final verdict?",
        answer:
          "No. The available lists are volunteer-run and unofficial; a listing is a warning that calls for checking, not a final judgement. The Ministry's lists and the Scientific Affairs decision prevail.",
      },
      {
        question: "What is a hijacked journal?",
        answer:
          "A legitimate journal whose name and ISSN are copied by fraudsters on a fake website that collects papers and fees. Papers sent to the fake site are not published in the real journal and are not indexed with it.",
      },
      {
        question: "Is a journal reputable if it appears in Google Scholar?",
        answer:
          "No. Google Scholar indexes whatever it finds on the web and does not assess journals. Reputation is judged by indexing in databases such as Scopus and Web of Science, verified at the source, and by the Ministry's instructions.",
      },
      {
        question: "Is the journal check free and open to everyone?",
        answer:
          "Yes. It works without signing in. Its result is advisory: it helps you verify, but it does not replace the decision of Scientific Affairs.",
      },
    ],
  },
  "warning-signs": {
    title: "Warning Signs of a Predatory Journal",
    description:
      "A practical checklist of the signs that expose predatory journals: instant acceptance, fees before review, misleading metrics and fake editorial boards.",
    navLabel: "Warning signs",
    intro:
      "No single sign proves anything, but several signs from this list together are reason enough to stop and check before you submit.",
    sections: [
      {
        heading: "Signs in the emails",
        paragraphs: ["Many predatory journals start with an unsolicited email:"],
        bullets: [
          "A flattering personal invitation to publish in a field far from yours.",
          "A promise of acceptance within days or publication within a week.",
          "A request for fees before review, or an acceptance letter with no reviewer comments.",
          "Free webmail addresses instead of an address on the publisher's domain.",
        ],
      },
      {
        heading: "Signs on the website",
        paragraphs: ["Read the journal's site critically:"],
        bullets: [
          "A very broad name spanning unrelated fields, or a near-copy of a well-known journal's name.",
          "Impact factors from unknown services (misleading metrics) instead of Journal Citation Reports or CiteScore.",
          "Claims of Scopus or Web of Science indexing that the databases themselves do not confirm.",
          "Editorial board members who cannot be verified, or known researchers who never agreed to serve.",
          "No clear policies on peer review, copyright, retraction and corrections.",
          "A postal address that is fake or does not match the country the journal claims.",
        ],
      },
      {
        heading: "Signs in the published papers",
        paragraphs: [
          "Read two or three recent papers: frequent language errors, unrelated topics side by side, weak methods and huge monthly issues all point to the absence of real review.",
        ],
      },
      {
        heading: "What to do if you see these signs",
        paragraphs: [
          "Do not submit and do not pay. Look the journal up in this guide's lists, run the journal check and talk to Scientific Affairs. If you have already submitted, withdraw the paper in writing before publication.",
        ],
      },
    ],
  },
  "hijacked-journals": {
    title: "Hijacked Journals: How to Spot a Fake Journal Website",
    description:
      "What hijacked journals are, how fraudsters clone a real journal's website, and practical steps to confirm the genuine site before you submit and pay.",
    navLabel: "Hijacked journals",
    intro:
      "A hijacked journal is a real, sometimes indexed, journal whose name and ISSN are copied by fraudsters on a fake website. Authors submit and pay the fake site, then find that their paper never appears in the real journal or in the indexes.",
    sections: [
      {
        heading: "How hijacking happens",
        paragraphs: [
          "Hijackers usually target older print journals or journals with a weak web presence. They register a domain close to the journal's name, copy its logo and details, and sometimes manipulate links on index pages so that they point to the fake site.",
        ],
      },
      {
        heading: "How to confirm the genuine site",
        paragraphs: ["Before you submit:"],
        bullets: [
          "Search the hijacked journals list in this guide; each page shows the fake site and, when known, the authentic journal and its site.",
          "Open the journal's record on the official Scopus or Web of Science site and compare the publisher and website.",
          "Check the ISSN on the ISSN Portal (portal.issn.org) and the publisher registered there.",
          "Look at the domain's registration date: a newly registered domain for an old journal is a red flag.",
          "Confirm that papers on the site actually appear in the index under the journal's name.",
        ],
      },
      {
        heading: "If you fell for a hijacked site",
        paragraphs: [
          "Keep all emails and payment receipts, notify the authentic journal and the index, and contact Scientific Affairs about the right procedure for your paper.",
        ],
      },
    ],
  },
  "verify-indexing": {
    title: "How to Verify Scopus and Web of Science Indexing Yourself",
    description:
      "Step-by-step checks that a journal is really indexed in Scopus and Web of Science, how to find its quartile, and why certificates on the journal's site are not enough.",
    navLabel: "Verify indexing",
    intro:
      "Indexing claims are the easiest thing to fake, so never rely on a badge or certificate on the journal's site. Check the official source yourself; it takes minutes.",
    sections: [
      {
        heading: "Checking Scopus",
        paragraphs: ["Scopus is owned by Elsevier, and its source list can be searched for free:"],
        bullets: [
          "Open the Sources page on the official Scopus site (scopus.com/sources) and search by title or ISSN.",
          "Make sure coverage is ongoing and includes your publication year; some titles have been discontinued.",
          "Compare the publisher and website on the source page with the site you are about to submit to.",
          "You can also consult the source list Elsevier publishes periodically.",
        ],
      },
      {
        heading: "Checking Web of Science",
        paragraphs: ["Clarivate owns Web of Science and offers a free tool to search covered journals:"],
        bullets: [
          "Open the Master Journal List on Clarivate's official site (mjl.clarivate.com) and search by title or ISSN.",
          "Note the index the journal appears in (such as SCIE, SSCI, AHCI or ESCI), since instructions may treat them differently.",
          "Compare the publisher and website with the site you are dealing with.",
        ],
      },
      {
        heading: "Finding the quartile",
        paragraphs: [
          "Quartiles are assigned per subject category, so a journal can be Q1 in one category and Q2 in another. Use the official data for the year your instructions require, and keep a dated screenshot of the journal's record.",
        ],
      },
      {
        heading: "A note about this portal",
        paragraphs: [
          "The university has no Scopus API access, so the journal check shows the Scopus status as \"unknown\"; that never means the journal is not indexed. Check Scopus manually with the steps above.",
        ],
      },
    ],
  },
  "predatory-conferences": {
    title: "Predatory Conferences: How to Recognise a Genuine Conference",
    description:
      "What predatory conferences are, their warning signs, and how to check a conference and where its papers are published before you register and pay.",
    navLabel: "Predatory conferences",
    intro:
      "Just as there are predatory journals, there are predatory conferences: events that collect high registration fees and promise publication in indexed journals, while review is a formality and the papers end up in non-reputable outlets.",
    sections: [
      {
        heading: "Signs of a predatory conference",
        paragraphs: ["Stop if you notice several of these:"],
        bullets: [
          "An organiser running dozens of conferences in unrelated fields in the same city on the same dates.",
          "Abstracts accepted within hours, with no real scientific committee.",
          "A promise that every paper will appear in a \"Scopus-indexed journal\" that is never named, or is not in Scopus.",
          "No known academic institution or scholarly society behind the event.",
          "\"Virtual attendance\" with a certificate but no actual presentation.",
        ],
      },
      {
        heading: "How to check a conference",
        paragraphs: [],
        bullets: [
          "Look up the organiser and past editions, and whether their papers really appeared where promised.",
          "If a journal is promised, run the journal check on that journal and verify its indexing.",
          "If proceedings go to a book series, verify the series in Scopus or Web of Science.",
          "Ask Scientific Affairs before registering if you will rely on the paper in an official procedure.",
        ],
      },
    ],
  },
};
