import {
  requestOperation,
  type ApiOperationId,
  type ApiRequestOptions,
} from "../generated/client";
import type { ProblemChangeTicket, TicketPage } from "../client";

export interface TicketContext {
  kind: "problem" | "exam" | "contest" | "mine" | "manage";
  id?: number;
  problemID?: number;
  targetScope?: string;
  teamProblemSetID?: number;
}
type TicketOperation =
  | "list"
  | "create"
  | "get"
  | "update"
  | "delete"
  | "reject"
  | "resolve"
  | "attachment";
const operations: Record<
  TicketContext["kind"],
  Record<TicketOperation, ApiOperationId>
> = {
  mine: {
    list: "listMyProblemChangeTickets",
    create: "createProblemChangeTicket",
    get: "getProblemChangeTicket",
    update: "updateProblemChangeTicket",
    delete: "deleteProblemChangeTicket",
    reject: "rejectProblemChangeTicket",
    resolve: "resolveProblemChangeTicket",
    attachment: "downloadTicketAttachment",
  },
  manage: {
    list: "listProblemChangeTickets",
    create: "createProblemChangeTicket",
    get: "getProblemChangeTicket",
    update: "updateProblemChangeTicket",
    delete: "deleteProblemChangeTicket",
    reject: "rejectProblemChangeTicket",
    resolve: "resolveProblemChangeTicket",
    attachment: "downloadTicketAttachment",
  },
  problem: {
    list: "listProblemFeedbackTickets",
    create: "createProblemFeedbackTicket",
    get: "getProblemFeedbackTicket",
    update: "updateProblemFeedbackTicket",
    delete: "deleteProblemFeedbackTicket",
    reject: "rejectProblemFeedbackTicket",
    resolve: "resolveProblemFeedbackTicket",
    attachment: "downloadProblemFeedbackTicketAttachment",
  },
  exam: {
    list: "listExamFeedbackTickets",
    create: "createExamFeedbackTicket",
    get: "getExamFeedbackTicket",
    update: "updateExamFeedbackTicket",
    delete: "deleteExamFeedbackTicket",
    reject: "rejectExamFeedbackTicket",
    resolve: "resolveExamFeedbackTicket",
    attachment: "downloadExamFeedbackTicketAttachment",
  },
  contest: {
    list: "listContestFeedbackTickets",
    create: "createContestFeedbackTicket",
    get: "getContestFeedbackTicket",
    update: "updateContestFeedbackTicket",
    delete: "deleteContestFeedbackTicket",
    reject: "rejectContestFeedbackTicket",
    resolve: "resolveContestFeedbackTicket",
    attachment: "downloadContestFeedbackTicketAttachment",
  },
};
export function ticketRequest<T = ProblemChangeTicket>(
  context: TicketContext,
  operation: TicketOperation,
  ticketID?: number,
  options: ApiRequestOptions = {},
) {
  return requestOperation<T>(operations[context.kind][operation], {
    ...options,
    path: {
      id:
        context.kind === "mine" || context.kind === "manage"
          ? ticketID || 0
          : context.id || 0,
      contest_id: context.id || 0,
      ticket_id: ticketID || 0,
      ...options.path,
    },
  });
}
export function listTickets(
  context: TicketContext,
  query: ApiRequestOptions["query"],
  signal?: AbortSignal,
) {
  return ticketRequest<TicketPage>(context, "list", undefined, {
    signal,
    query: {
      ...query,
      ...((context.kind === "exam" || context.kind === "contest") &&
      context.problemID
        ? { problem_id: context.problemID }
        : {}),
    },
  });
}
