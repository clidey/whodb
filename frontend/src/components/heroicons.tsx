/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Centralized heroicons imports from @heroicons/react
// This allows for easy override by extensions

import {
  AdjustmentsHorizontalIcon as AdjustmentsHorizontalIconCE,
  ChevronUpDownIcon as ChevronUpDownIconCE,
  ArrowDownCircleIcon as ArrowDownCircleIconCE,
  ArrowDownTrayIcon as ArrowDownTrayIconCE,
  ArrowLeftStartOnRectangleIcon as ArrowLeftStartOnRectangleIconCE,
  ArrowPathIcon as ArrowPathIconCE,
  ArrowPathRoundedSquareIcon as ArrowPathRoundedSquareIconCE,
  ArrowTopRightOnSquareIcon as ArrowTopRightOnSquareIconCE,
  ArrowUpCircleIcon as ArrowUpCircleIconCE,
  BanknotesIcon as BanknotesIconCE,
  BellAlertIcon as BellAlertIconCE,
  BookOpenIcon as BookOpenIconCE,
  BuildingStorefrontIcon as BuildingStorefrontIconCE,
  CalculatorIcon as CalculatorIconCE,
  CalendarIcon as CalendarIconCE,
  ChartBarIcon as ChartBarIconCE,
  ChatBubbleLeftRightIcon as ChatBubbleLeftRightIconCE,
  CheckCircleIcon as CheckCircleIconCE,
  ChevronDownIcon as ChevronDownIconCE,
  ChevronRightIcon as ChevronRightIconCE,
  ChevronUpIcon as ChevronUpIconCE,
  CircleStackIcon as CircleStackIconCE,
  ClipboardDocumentIcon as ClipboardDocumentIconCE,
  CloudIcon as CloudIconCE,
  ClipboardIcon as ClipboardIconCE,
  ClockIcon as ClockIconCE,
  CogIcon as CogIconCE,
  CodeBracketIcon as CodeBracketIconCE,
  CommandLineIcon as CommandLineIconCE,
  CursorArrowRaysIcon as CursorArrowRaysIconCE,
  DocumentDuplicateIcon as DocumentDuplicateIconCE,
  DocumentIcon as DocumentIconCE,
  DocumentTextIcon as DocumentTextIconCE,
  EllipsisHorizontalIcon as EllipsisHorizontalIconCE,
  EllipsisVerticalIcon as EllipsisVerticalIconCE,
  EnvelopeIcon as EnvelopeIconCE,
  ExclamationCircleIcon as ExclamationCircleIconCE,
  EyeIcon as EyeIconCE,
  EyeSlashIcon as EyeSlashIconCE,
  FolderIcon as FolderIconCE,
  GlobeAltIcon as GlobeAltIconCE,
  HashtagIcon as HashtagIconCE,
  HomeIcon as HomeIconCE,
  InformationCircleIcon as InformationCircleIconCE,
  KeyIcon as KeyIconCE,
  LinkIcon as LinkIconCE,
  LockClosedIcon as LockClosedIconCE,
  ListBulletIcon as ListBulletIconCE,
  MagnifyingGlassIcon as MagnifyingGlassIconCE,
  PencilIcon as PencilIconCE,
  PencilSquareIcon as PencilSquareIconCE,
  PlayIcon as PlayIconCE,
  PlusCircleIcon as PlusCircleIconCE,
  PlusIcon as PlusIconCE,
  PresentationChartBarIcon as PresentationChartBarIconCE,
  PresentationChartLineIcon as PresentationChartLineIconCE,
  QuestionMarkCircleIcon as QuestionMarkCircleIconCE,
  RectangleGroupIcon as RectangleGroupIconCE,
  ShareIcon as ShareIconCE,
  ShieldCheckIcon as ShieldCheckIconCE,
  ShoppingBagIcon as ShoppingBagIconCE,
  SparklesIcon as SparklesIconCE,
  StarIcon as StarIconCE,
  TableCellsIcon as TableCellsIconCE,
  TrashIcon as TrashIconCE,
  UserGroupIcon as UserGroupIconCE,
  UsersIcon as UsersIconCE,
  XCircleIcon as XCircleIconCE,
  XMarkIcon as XMarkIconCE,
  AdjustmentsVerticalIcon as AdjustmentsVerticalIconCE,
  Bars3Icon as Bars3IconCE,
  Squares2X2Icon as Squares2X2IconCE,
} from '@heroicons/react/24/outline';

// Extensions may provide overrides for selected icons via registerHeroiconOverrides().
let iconOverrides: Record<string, any> = {};

/** Register heroicon overrides. */
export function registerHeroiconOverrides(overrides: Record<string, any>) {
    iconOverrides = overrides;
}

export const AdjustmentsHorizontalIcon = iconOverrides.AdjustmentsHorizontalIcon ?? AdjustmentsHorizontalIconCE;
export const ChevronUpDownIcon = iconOverrides.ChevronUpDownIcon ?? ChevronUpDownIconCE;
export const ArrowDownCircleIcon = iconOverrides.ArrowDownCircleIcon ?? ArrowDownCircleIconCE;
export const ArrowDownTrayIcon = iconOverrides.ArrowDownTrayIcon ?? ArrowDownTrayIconCE;
export const ArrowLeftStartOnRectangleIcon = iconOverrides.ArrowLeftStartOnRectangleIcon ?? ArrowLeftStartOnRectangleIconCE;
export const ArrowPathIcon = iconOverrides.ArrowPathIcon ?? ArrowPathIconCE;
export const ArrowPathRoundedSquareIcon = iconOverrides.ArrowPathRoundedSquareIcon ?? ArrowPathRoundedSquareIconCE;
export const ArrowTopRightOnSquareIcon = iconOverrides.ArrowTopRightOnSquareIcon ?? ArrowTopRightOnSquareIconCE;
export const ArrowUpCircleIcon = iconOverrides.ArrowUpCircleIcon ?? ArrowUpCircleIconCE;
export const BanknotesIcon = iconOverrides.BanknotesIcon ?? BanknotesIconCE;
export const BellAlertIcon = iconOverrides.BellAlertIcon ?? BellAlertIconCE;
export const BookOpenIcon = iconOverrides.BookOpenIcon ?? BookOpenIconCE;
export const BuildingStorefrontIcon = iconOverrides.BuildingStorefrontIcon ?? BuildingStorefrontIconCE;
export const CalculatorIcon = iconOverrides.CalculatorIcon ?? CalculatorIconCE;
export const CalendarIcon = iconOverrides.CalendarIcon ?? CalendarIconCE;
export const ChartBarIcon = iconOverrides.ChartBarIcon ?? ChartBarIconCE;
export const ChatBubbleLeftRightIcon = iconOverrides.ChatBubbleLeftRightIcon ?? ChatBubbleLeftRightIconCE;
export const CheckCircleIcon = iconOverrides.CheckCircleIcon ?? CheckCircleIconCE;
export const ChevronDownIcon = iconOverrides.ChevronDownIcon ?? ChevronDownIconCE;
export const ChevronRightIcon = iconOverrides.ChevronRightIcon ?? ChevronRightIconCE;
export const ChevronUpIcon = iconOverrides.ChevronUpIcon ?? ChevronUpIconCE;
export const CircleStackIcon = iconOverrides.CircleStackIcon ?? CircleStackIconCE;
export const ClipboardDocumentIcon = iconOverrides.ClipboardDocumentIcon ?? ClipboardDocumentIconCE;
export const CloudIcon = iconOverrides.CloudIcon ?? CloudIconCE;
export const ClipboardIcon = iconOverrides.ClipboardIcon ?? ClipboardIconCE;
export const ClockIcon = iconOverrides.ClockIcon ?? ClockIconCE;
export const CogIcon = iconOverrides.CogIcon ?? CogIconCE;
export const CodeBracketIcon = iconOverrides.CodeBracketIcon ?? CodeBracketIconCE;
export const CommandLineIcon = iconOverrides.CommandLineIcon ?? CommandLineIconCE;
export const CursorArrowRaysIcon = iconOverrides.CursorArrowRaysIcon ?? CursorArrowRaysIconCE;
export const DocumentDuplicateIcon = iconOverrides.DocumentDuplicateIcon ?? DocumentDuplicateIconCE;
export const DocumentIcon = iconOverrides.DocumentIcon ?? DocumentIconCE;
export const DocumentTextIcon = iconOverrides.DocumentTextIcon ?? DocumentTextIconCE;
export const EllipsisHorizontalIcon = iconOverrides.EllipsisHorizontalIcon ?? EllipsisHorizontalIconCE;
export const EllipsisVerticalIcon = iconOverrides.EllipsisVerticalIcon ?? EllipsisVerticalIconCE;
export const EnvelopeIcon = iconOverrides.EnvelopeIcon ?? EnvelopeIconCE;
export const ExclamationCircleIcon = iconOverrides.ExclamationCircleIcon ?? ExclamationCircleIconCE;
export const EyeIcon = iconOverrides.EyeIcon ?? EyeIconCE;
export const EyeSlashIcon = iconOverrides.EyeSlashIcon ?? EyeSlashIconCE;
export const FolderIcon = iconOverrides.FolderIcon ?? FolderIconCE;
export const GlobeAltIcon = iconOverrides.GlobeAltIcon ?? GlobeAltIconCE;
export const HashtagIcon = iconOverrides.HashtagIcon ?? HashtagIconCE;
export const HomeIcon = iconOverrides.HomeIcon ?? HomeIconCE;
export const InformationCircleIcon = iconOverrides.InformationCircleIcon ?? InformationCircleIconCE;
export const KeyIcon = iconOverrides.KeyIcon ?? KeyIconCE;
export const LinkIcon = iconOverrides.LinkIcon ?? LinkIconCE;
export const LockClosedIcon = iconOverrides.LockClosedIcon ?? LockClosedIconCE;
export const ListBulletIcon = iconOverrides.ListBulletIcon ?? ListBulletIconCE;
export const MagnifyingGlassIcon = iconOverrides.MagnifyingGlassIcon ?? MagnifyingGlassIconCE;
export const PencilIcon = iconOverrides.PencilIcon ?? PencilIconCE;
export const PencilSquareIcon = iconOverrides.PencilSquareIcon ?? PencilSquareIconCE;
export const PlayIcon = iconOverrides.PlayIcon ?? PlayIconCE;
export const PlusCircleIcon = iconOverrides.PlusCircleIcon ?? PlusCircleIconCE;
export const PlusIcon = iconOverrides.PlusIcon ?? PlusIconCE;
export const PresentationChartBarIcon = iconOverrides.PresentationChartBarIcon ?? PresentationChartBarIconCE;
export const PresentationChartLineIcon = iconOverrides.PresentationChartLineIcon ?? PresentationChartLineIconCE;
export const QuestionMarkCircleIcon = iconOverrides.QuestionMarkCircleIcon ?? QuestionMarkCircleIconCE;
export const RectangleGroupIcon = iconOverrides.RectangleGroupIcon ?? RectangleGroupIconCE;
export const ShareIcon = iconOverrides.ShareIcon ?? ShareIconCE;
export const ShieldCheckIcon = iconOverrides.ShieldCheckIcon ?? ShieldCheckIconCE;
export const ShoppingBagIcon = iconOverrides.ShoppingBagIcon ?? ShoppingBagIconCE;
export const SparklesIcon = iconOverrides.SparklesIcon ?? SparklesIconCE;
export const StarIcon = iconOverrides.StarIcon ?? StarIconCE;
export const TableCellsIcon = iconOverrides.TableCellsIcon ?? TableCellsIconCE;
export const TrashIcon = iconOverrides.TrashIcon ?? TrashIconCE;
export const UserGroupIcon = iconOverrides.UserGroupIcon ?? UserGroupIconCE;
export const UsersIcon = iconOverrides.UsersIcon ?? UsersIconCE;
export const XCircleIcon = iconOverrides.XCircleIcon ?? XCircleIconCE;
export const XMarkIcon = iconOverrides.XMarkIcon ?? XMarkIconCE;
export const AdjustmentsVerticalIcon = iconOverrides.AdjustmentsVerticalIcon ?? AdjustmentsVerticalIconCE;
export const Bars3Icon = iconOverrides.Bars3Icon ?? Bars3IconCE;
export const Squares2X2Icon = iconOverrides.Squares2X2Icon ?? Squares2X2IconCE;