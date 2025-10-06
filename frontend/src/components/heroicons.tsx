/**
 * Copyright 2025 Clidey, Inc.
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
// This allows for easy override in EE version

import {
  AdjustmentsHorizontalIcon as AdjustmentsHorizontalIconCE,
  ArrowDownCircleIcon as ArrowDownCircleIconCE,
  ArrowDownTrayIcon as ArrowDownTrayIconCE,
  ArrowLeftStartOnRectangleIcon as ArrowLeftStartOnRectangleIconCE,
  ArrowPathIcon as ArrowPathIconCE,
  ArrowPathRoundedSquareIcon as ArrowPathRoundedSquareIconCE,
  ArrowTopRightOnSquareIcon as ArrowTopRightOnSquareIconCE,
  ArrowUpCircleIcon as ArrowUpCircleIconCE,
  BanknotesIcon as BanknotesIconCE,
  BellAlertIcon as BellAlertIconCE,
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
} from '@heroicons/react/24/outline';

// EE may provide overrides for selected icons. In CE builds this should resolve
// to a stub that exports a default empty object.
// eslint-disable-next-line @typescript-eslint/consistent-type-imports
import EEOverridesDefault from '@ee/heroicons';
const EEOverrides: Record<string, any> = (EEOverridesDefault as any) || {};

export const AdjustmentsHorizontalIcon = EEOverrides.AdjustmentsHorizontalIcon ?? AdjustmentsHorizontalIconCE;
export const ArrowDownCircleIcon = EEOverrides.ArrowDownCircleIcon ?? ArrowDownCircleIconCE;
export const ArrowDownTrayIcon = EEOverrides.ArrowDownTrayIcon ?? ArrowDownTrayIconCE;
export const ArrowLeftStartOnRectangleIcon = EEOverrides.ArrowLeftStartOnRectangleIcon ?? ArrowLeftStartOnRectangleIconCE;
export const ArrowPathIcon = EEOverrides.ArrowPathIcon ?? ArrowPathIconCE;
export const ArrowPathRoundedSquareIcon = EEOverrides.ArrowPathRoundedSquareIcon ?? ArrowPathRoundedSquareIconCE;
export const ArrowTopRightOnSquareIcon = EEOverrides.ArrowTopRightOnSquareIcon ?? ArrowTopRightOnSquareIconCE;
export const ArrowUpCircleIcon = EEOverrides.ArrowUpCircleIcon ?? ArrowUpCircleIconCE;
export const BanknotesIcon = EEOverrides.BanknotesIcon ?? BanknotesIconCE;
export const BellAlertIcon = EEOverrides.BellAlertIcon ?? BellAlertIconCE;
export const BuildingStorefrontIcon = EEOverrides.BuildingStorefrontIcon ?? BuildingStorefrontIconCE;
export const CalculatorIcon = EEOverrides.CalculatorIcon ?? CalculatorIconCE;
export const CalendarIcon = EEOverrides.CalendarIcon ?? CalendarIconCE;
export const ChartBarIcon = EEOverrides.ChartBarIcon ?? ChartBarIconCE;
export const ChatBubbleLeftRightIcon = EEOverrides.ChatBubbleLeftRightIcon ?? ChatBubbleLeftRightIconCE;
export const CheckCircleIcon = EEOverrides.CheckCircleIcon ?? CheckCircleIconCE;
export const ChevronDownIcon = EEOverrides.ChevronDownIcon ?? ChevronDownIconCE;
export const ChevronRightIcon = EEOverrides.ChevronRightIcon ?? ChevronRightIconCE;
export const ChevronUpIcon = EEOverrides.ChevronUpIcon ?? ChevronUpIconCE;
export const CircleStackIcon = EEOverrides.CircleStackIcon ?? CircleStackIconCE;
export const ClipboardDocumentIcon = EEOverrides.ClipboardDocumentIcon ?? ClipboardDocumentIconCE;
export const ClipboardIcon = EEOverrides.ClipboardIcon ?? ClipboardIconCE;
export const ClockIcon = EEOverrides.ClockIcon ?? ClockIconCE;
export const CogIcon = EEOverrides.CogIcon ?? CogIconCE;
export const CodeBracketIcon = EEOverrides.CodeBracketIcon ?? CodeBracketIconCE;
export const CommandLineIcon = EEOverrides.CommandLineIcon ?? CommandLineIconCE;
export const CursorArrowRaysIcon = EEOverrides.CursorArrowRaysIcon ?? CursorArrowRaysIconCE;
export const DocumentDuplicateIcon = EEOverrides.DocumentDuplicateIcon ?? DocumentDuplicateIconCE;
export const DocumentIcon = EEOverrides.DocumentIcon ?? DocumentIconCE;
export const DocumentTextIcon = EEOverrides.DocumentTextIcon ?? DocumentTextIconCE;
export const EllipsisHorizontalIcon = EEOverrides.EllipsisHorizontalIcon ?? EllipsisHorizontalIconCE;
export const EllipsisVerticalIcon = EEOverrides.EllipsisVerticalIcon ?? EllipsisVerticalIconCE;
export const EnvelopeIcon = EEOverrides.EnvelopeIcon ?? EnvelopeIconCE;
export const EyeIcon = EEOverrides.EyeIcon ?? EyeIconCE;
export const EyeSlashIcon = EEOverrides.EyeSlashIcon ?? EyeSlashIconCE;
export const FolderIcon = EEOverrides.FolderIcon ?? FolderIconCE;
export const GlobeAltIcon = EEOverrides.GlobeAltIcon ?? GlobeAltIconCE;
export const HashtagIcon = EEOverrides.HashtagIcon ?? HashtagIconCE;
export const HomeIcon = EEOverrides.HomeIcon ?? HomeIconCE;
export const InformationCircleIcon = EEOverrides.InformationCircleIcon ?? InformationCircleIconCE;
export const KeyIcon = EEOverrides.KeyIcon ?? KeyIconCE;
export const LinkIcon = EEOverrides.LinkIcon ?? LinkIconCE;
export const LockClosedIcon = EEOverrides.LockClosedIcon ?? LockClosedIconCE;
export const ListBulletIcon = EEOverrides.ListBulletIcon ?? ListBulletIconCE;
export const MagnifyingGlassIcon = EEOverrides.MagnifyingGlassIcon ?? MagnifyingGlassIconCE;
export const PencilIcon = EEOverrides.PencilIcon ?? PencilIconCE;
export const PencilSquareIcon = EEOverrides.PencilSquareIcon ?? PencilSquareIconCE;
export const PlayIcon = EEOverrides.PlayIcon ?? PlayIconCE;
export const PlusCircleIcon = EEOverrides.PlusCircleIcon ?? PlusCircleIconCE;
export const PlusIcon = EEOverrides.PlusIcon ?? PlusIconCE;
export const PresentationChartBarIcon = EEOverrides.PresentationChartBarIcon ?? PresentationChartBarIconCE;
export const PresentationChartLineIcon = EEOverrides.PresentationChartLineIcon ?? PresentationChartLineIconCE;
export const QuestionMarkCircleIcon = EEOverrides.QuestionMarkCircleIcon ?? QuestionMarkCircleIconCE;
export const RectangleGroupIcon = EEOverrides.RectangleGroupIcon ?? RectangleGroupIconCE;
export const ShareIcon = EEOverrides.ShareIcon ?? ShareIconCE;
export const ShoppingBagIcon = EEOverrides.ShoppingBagIcon ?? ShoppingBagIconCE;
export const SparklesIcon = EEOverrides.SparklesIcon ?? SparklesIconCE;
export const StarIcon = EEOverrides.StarIcon ?? StarIconCE;
export const TableCellsIcon = EEOverrides.TableCellsIcon ?? TableCellsIconCE;
export const TrashIcon = EEOverrides.TrashIcon ?? TrashIconCE;
export const UserGroupIcon = EEOverrides.UserGroupIcon ?? UserGroupIconCE;
export const UsersIcon = EEOverrides.UsersIcon ?? UsersIconCE;
export const XCircleIcon = EEOverrides.XCircleIcon ?? XCircleIconCE;
export const XMarkIcon = EEOverrides.XMarkIcon ?? XMarkIconCE;
export const AdjustmentsVerticalIcon = EEOverrides.AdjustmentsVerticalIcon ?? AdjustmentsVerticalIconCE;